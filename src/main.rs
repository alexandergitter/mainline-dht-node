mod bencode;
mod fixed_vec;

use bencode::{Bencode, Dict};
use cursive::view::*;
use cursive::views::*;
use cursive::Cursive;
use fixed_vec::FixedVec;
use rand::prelude::*;
use std::collections::VecDeque;
use std::mem::MaybeUninit;
use std::net::{SocketAddrV4, ToSocketAddrs, UdpSocket};
use std::ops::{Index, IndexMut, Range};
use std::str;
use std::sync::mpsc;
use std::thread;
use std::time::{Duration, Instant, SystemTime};

#[derive(Debug)]
struct MyInfo {
    client_version: Option<String>,
    id: NodeId,
    port: u16,
}

enum NetCommand {
    Bootstrap(SocketAddrV4),
}

const BOOTSTRAP_NODE: &str = "router.bittorrent.com:6881";

fn main() {
    let (gui_tx, net_rx) = mpsc::channel::<NetCommand>();
    let (net_tx, gui_rx) = mpsc::channel::<Box<std::fmt::Display + Send>>();

    thread::spawn(move || {
        let mut my_node_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut my_node_id);

        let my_info = MyInfo {
            client_version: None,
            id: my_node_id,
            port: 6881,
        };

        let socket = UdpSocket::bind("0.0.0.0:12345").unwrap();
        socket.set_nonblocking(true).unwrap();

        let mut buf = vec![0; 1024];

        loop {
            if let Ok(command) = net_rx.try_recv() {
                match command {
                    NetCommand::Bootstrap(addr) => {
                        let mut args = Dict::new();
                        args.insert(b"id".to_vec(), Bencode::Bytes(my_info.id.to_vec()));
                        args.insert(b"target".to_vec(), Bencode::Bytes(my_info.id.to_vec()));

                        let req = krpc_request("find_node", args, &my_info);

                        net_tx.send(Box::new(format!("sending: {}", req)));

                        let res = socket.send_to(&req.encode(), addr);
                        if let Err(_) = res {
                            net_tx.send(Box::new("error sending"));
                        }
                    }
                }
            }

            match socket.recv(&mut buf) {
                Ok(num) => {
                    let result = Bencode::decode(&buf[..num]).unwrap();
                    net_tx.send(Box::new(format!("received: {}", result)));
                }
                Err(_) => {}
            };

            thread::sleep(Duration::from_millis(100));
        }
    });

    // Creates the cursive root - required for every application.
    let mut cs = Cursive::default();

    cs.set_user_data(gui_tx);

    cs.add_fullscreen_layer(
        LinearLayout::vertical()
            .child(BoxView::with_full_width(Panel::new(TextView::new(
                "b - bootstrap    i - check inbox",
            ))))
            .child(BoxView::with_full_height(
                Panel::new(
                    ScrollView::new(
                        TextView::new("Hello Dialog as das da sd asd as d asd!")
                            .with_id("main_content"),
                    )
                    .scroll_strategy(ScrollStrategy::StickToBottom),
                )
                .title("Hello hello"),
            )),
    );

    cs.add_global_callback('b', |cs| {
        cs.add_layer(
            Dialog::new()
                .title("Bootstrap node (ip:port)")
                .padding((1, 1, 1, 0))
                .content(
                    EditView::new()
                        .content(BOOTSTRAP_NODE)
                        .on_submit(|cs, val| {
                            let tx = cs.user_data::<mpsc::Sender<NetCommand>>().unwrap();
                            tx.send(NetCommand::Bootstrap(val.parse().unwrap()));

                            cs.pop_layer();
                        }),
                ),
        );
    });

    cs.set_fps(30);

    while cs.is_running() {
        if let Ok(msg) = gui_rx.try_recv() {
            cs.call_on_id("main_content", |view: &mut TextView| {
                let new_content = format!("\n===\n{}", msg);
                view.append(new_content)
            });
        }

        cs.step();
    }
}

fn krpc_request(method: &str, args: Dict, my_info: &MyInfo) -> Bencode {
    let mut transaction_id = [0u8; 2];
    rand::thread_rng().fill_bytes(&mut transaction_id);

    let mut dict = Dict::new();
    dict.insert(b"t".to_vec(), Bencode::Bytes(transaction_id.to_vec()));
    dict.insert(b"y".to_vec(), Bencode::Bytes(b"q".to_vec()));
    dict.insert(b"q".to_vec(), Bencode::Bytes(method.as_bytes().to_owned()));
    dict.insert(b"a".to_vec(), Bencode::Dict(args));

    if let Some(ref client_version) = my_info.client_version {
        dict.insert(
            b"v".to_vec(),
            Bencode::Bytes(client_version.as_bytes().to_owned()),
        );
    }

    Bencode::Dict(dict)
}

enum KrpcError {
    SendError,
    ResponseTimeout,
    MalformedResponse,
    InvalidResponse,
    ErrorResponse,
}

type NodeId = [u8; 20];

#[derive(Debug, PartialEq, Clone)]
struct NodeContactInfo {
    id: NodeId,
    address: SocketAddrV4,
}

#[derive(PartialEq, Debug)]
enum NodeRating {
    Good,
    Questionable,
    Bad,
}

#[derive(Debug, PartialEq, Clone)]
struct RoutingEntry {
    node: NodeContactInfo,
    last_response: Option<Instant>,
    last_query: Option<Instant>,
}

impl RoutingEntry {
    fn new(node: NodeContactInfo) -> RoutingEntry {
        RoutingEntry {
            node,
            last_query: None,
            last_response: None,
        }
    }

    fn rating(&self) -> NodeRating {
        let minutes_since_last_query = self
            .last_query
            .map(|instant| instant.elapsed().as_secs() / 60);
        let minutes_since_last_response = self
            .last_response
            .map(|instant| instant.elapsed().as_secs() / 60);

        match (minutes_since_last_query, minutes_since_last_response) {
            // Node has responded in the last 15 minutes
            (_, Some(r)) if r <= 15 => NodeRating::Good,
            // Node has responded at least once and sent us a query in the last 15 minutes
            (Some(q), Some(_)) if q <= 15 => NodeRating::Good,
            // We haven't heard anything from the node yet (likely was a referral)
            (None, None) => NodeRating::Questionable,
            // TODO: after 15 minutes of inactivity, nodes should not become bad, but
            //       questionable and we should try pinging them before marking them bad
            _ => NodeRating::Bad,
        }
    }

    fn update(&mut self, seen_in: SeenIn) {
        match seen_in {
            SeenIn::Query => self.last_query = Some(Instant::now()),
            SeenIn::Response => self.last_response = Some(Instant::now()),
            SeenIn::Referral => {}
        }
    }
}

#[derive(Debug)]
struct Bucket {
    entries: FixedVec<RoutingEntry>,
    bounds: Range<u32>,
}

impl Bucket {
    fn new(bounds: Range<u32>) -> Bucket {
        Bucket {
            entries: FixedVec::new(),
            bounds,
        }
    }
}

struct RoutingTable {
    buckets: VecDeque<Bucket>,
    reference_id: NodeId,
}

enum SeenIn {
    Query,
    Response,
    Referral,
}

impl RoutingTable {
    fn new(reference_id: NodeId) -> RoutingTable {
        let initial_bucket = Bucket::new(0..160);

        // TODO: figure out the avg. number of nodes added on bootstrap and set initial size accordingly
        let mut buckets = VecDeque::new();
        buckets.push_back(initial_bucket);

        RoutingTable {
            buckets,
            reference_id,
        }
    }

    fn find_node(&self, node_id: &NodeId) -> Option<&NodeContactInfo> {
        let prefix_len = node_id::common_prefix_length(&self.reference_id, &node_id);
        // TODO: this currently does a linear search; it should probably be done
        //       with a binary search, a B+ index tree, or something else
        let bucket = self
            .buckets
            .iter()
            .find(|bucket| bucket.bounds.contains(&prefix_len))
            .expect(&format!(
                "no bucket for prefix length {} - this should never happen",
                prefix_len
            ));

        bucket
            .entries
            .iter()
            .find(|entry| &entry.node.id == node_id)
            .map(|entry| &entry.node)
    }

    fn find_closest(&self, node_id: &NodeId) -> FixedVec<&NodeContactInfo> {
        let prefix_len = node_id::common_prefix_length(&self.reference_id, &node_id);
        // TODO: this currently does a linear search; it should probably be done
        //       with a binary search, a B+ index tree, or something else
        let (bucket_index, bucket) = self
            .buckets
            .iter()
            .enumerate()
            .find(|(_, bucket)| bucket.bounds.contains(&prefix_len))
            .expect(&format!(
                "no bucket for prefix length {} - this should never happen",
                prefix_len
            ));

        let mut result = FixedVec::<&NodeContactInfo>::new();

        for entry in bucket.entries.iter() {
            result.push(&entry.node);
        }

        let mut offset = 1;
        while result.is_not_full() {
            if bucket_index >= offset {
                let lower_bucket = self.buckets.get(bucket_index - offset).expect("There should be a valid bucket here. Is the `if` guard broken?");

                for entry in lower_bucket.entries.iter() {
                    result.push(&entry.node);
                    if result.is_full() {
                        break;
                    }
                }
            }

            if ((bucket_index + offset) < self.buckets.len()) && (result.is_not_full()) {
                let higher_bucket = self.buckets.get(bucket_index +  offset).expect("There should be a valid bucket here. Is the `if` guard broken?");

                for entry in higher_bucket.entries.iter() {
                    result.push(&entry.node);
                    if result.is_full() {
                        break;
                    }
                }
            }

            offset += 1;
        }

        result
    }

    fn update(&mut self, node: NodeContactInfo, seen_in: SeenIn) {
        assert!(!self.buckets.is_empty(), "Routing table contains no buckets. It must always have at least one (full-range) bucket.");

        /* Go through buckets and get the one corresponding to the prefix */
        // TODO: this currently does a linear search; it should probably be done
        //       with a binary search, a B+ index tree, or something else
        let prefix_len = node_id::common_prefix_length(&self.reference_id, &node.id);
        let (bucket_index, bucket) = self
            .buckets
            .iter_mut()
            .enumerate()
            .find(|(_, bucket)| bucket.bounds.contains(&prefix_len))
            .expect(&format!(
                "no bucket for prefix length {} - this should never happen",
                prefix_len
            ));

        /* See if we already have an entry for this node */
        let entry_position = bucket
            .entries
            .iter()
            .position(|entry| entry.node.id == node.id);

        if let Some(entry_position) = entry_position {
            bucket.entries[entry_position].update(seen_in);
            return;
        }

        /* Bucket doesn't have the node yet. Check if we can insert it. */
        let mut new_entry = RoutingEntry::new(node);
        new_entry.update(seen_in);

        /* Case 1: bucket still has open slots */
        if bucket.entries.is_not_full() {
            bucket.entries.push(new_entry);
            return;
        }

        /* Case 2: bucket has a bad node that can be replaced */
        // TODO: questionable nodes should be pinged to determine whether
        //       they are bad and can be replaced. Maybe add a "cache" area
        //       to buckets, so they can remember new candidates they can
        //       swap in once an entry becomes bad?
        let bad_node = bucket
            .entries
            .iter()
            .position(|entry| entry.rating() == NodeRating::Bad);
        if let Some(bad_node) = bad_node {
            bucket.entries[bad_node] = new_entry;
            return;
        }

        /* Case 3: bucket can be split */
        if (bucket.bounds.end - bucket.bounds.start) > 1 {
            let lower_bounds = Range {
                start: bucket.bounds.start,
                // the lower bucket will only cover one prefix bit
                end: bucket.bounds.start + 1,
            };
            let upper_bounds = Range {
                start: bucket.bounds.start + 1,
                end: bucket.bounds.end,
            };

            /* Drain nodes that fall into the upper bucket */
            let mut i = 0;
            let mut drained_upper_nodes = FixedVec::new();
            while i != bucket.entries.len() {
                let prefix_len =
                    node_id::common_prefix_length(&self.reference_id, &bucket.entries[i].node.id);
                if upper_bounds.contains(&prefix_len) {
                    drained_upper_nodes.push(bucket.entries.swap_remove(i));
                } else {
                    i += 1;
                }
            }

            // TODO: Correctly handle the edge case that all existing nodes AND the
            //       new node fall into the same bucket:
            //       1. The bucket with all nodes can be split further
            //          -> either continue splitting one by one at the far end
            //             OR to avoid this situation entirely, sort existing + new
            //             entries by distance from reference and split in the middle
            //       2. The bucket with all nodes cannot be split further
            //          -> split done, node will be discarded
            let lower_has_slot = bucket.entries.is_not_full();
            let upper_has_slot = drained_upper_nodes.is_not_full();

            if lower_has_slot && lower_bounds.contains(&prefix_len) {
                bucket.bounds = lower_bounds;
                let mut upper_bucket = Bucket::new(upper_bounds);
                upper_bucket.entries.append(drained_upper_nodes);

                bucket.entries.push(new_entry);

                self.buckets.insert(bucket_index + 1, upper_bucket);
            } else if upper_has_slot && upper_bounds.contains(&prefix_len) {
                bucket.bounds = lower_bounds;
                let mut upper_bucket = Bucket::new(upper_bounds);
                upper_bucket.entries.append(drained_upper_nodes);

                upper_bucket.entries.push(new_entry);

                self.buckets.insert(bucket_index + 1, upper_bucket);
            } else {
                // We decided not to split, revert draining some nodes from bucket
                bucket.entries.append(drained_upper_nodes);
            }
        }
    }
}

mod node_id {
    use crate::NodeId;

    pub fn from_slice(s: &[u8]) -> Result<NodeId, &'static str> {
        if s.len() != 20 {
            return Err("Invalid node id - must be 20 bytes long");
        }

        let mut arr = [0u8; 20];
        arr.copy_from_slice(s);

        Ok(arr)
    }

    pub fn from_vec(v: Vec<u8>) -> Result<NodeId, &'static str> {
        from_slice(v.as_slice())
    }

    pub fn multiple_from_vec(v: Vec<u8>) -> Result<Vec<NodeId>, &'static str> {
        v.chunks_exact(20).map(|c| from_slice(c)).collect()
    }

    pub fn distance(a: &NodeId, b: &NodeId) -> [u8; 20] {
        let mut result = [0u8; 20];

        for i in 0..result.len() {
            result[i] = a[i] ^ b[i];
        }

        result
    }

    pub fn common_prefix_length(a: &NodeId, b: &NodeId) -> u32 {
        let distance = distance(a, b);
        let mut result = 0;

        for byte in &distance {
            result += byte.leading_zeros();

            if byte.leading_zeros() < 8 {
                break;
            }
        }

        result
    }
}

#[derive(Debug)]
struct FindNodeResponse {
    transaction_id: Vec<u8>,
    client_version: Option<String>,
    responder_id: NodeId,
    node_ids: Vec<NodeId>,
}

impl FindNodeResponse {
    fn from_bencode(mut dict: Dict) -> Result<FindNodeResponse, &'static str> {
        let transaction_id = match dict.remove::<[u8]>(b"t") {
            Some(Bencode::Bytes(v)) => v,
            _ => return Err("No or invalid transaction id"),
        };

        let client_version = match dict.remove::<[u8]>(b"v") {
            Some(Bencode::Bytes(v)) => Some(String::from_utf8_lossy(&v).into_owned()),
            _ => None,
        };

        let mut return_values = match dict.remove::<[u8]>(b"r") {
            Some(Bencode::Dict(v)) => v,
            _ => return Err("No or invalid return value dict"),
        };

        let responder_id = match return_values.remove::<[u8]>(b"id") {
            Some(Bencode::Bytes(v)) => node_id::from_vec(v)?,
            _ => return Err("No or invalid responder id"),
        };

        let node_ids = match return_values.remove::<[u8]>(b"nodes") {
            Some(Bencode::Bytes(v)) => node_id::multiple_from_vec(v)?,
            _ => return Err("No or invalid node ids"),
        };

        Ok(FindNodeResponse {
            transaction_id,
            client_version,
            responder_id,
            node_ids,
        })
    }

    fn transaction_id(&self) -> &Vec<u8> {
        &self.transaction_id
    }
}

fn krpc_handle(bencode: Bencode) -> Result<(), &'static str> {
    let msg = match bencode {
        Bencode::Dict(map) => map,
        _ => return Err("KRPC message is not a Dict"),
    };

    let y = match msg.get(b"y".as_ref()) {
        Some(Bencode::Bytes(v)) => v,
        _ => return Err("KRPC message does not contain key y or invalid value type"),
    };

    match &y[..] {
        b"r" => krpc_on_return(msg),
        b"e" => krpc_on_error(msg),
        _ => Err("Unhandled KRPC message type"),
    }
}

fn krpc_on_return(msg: Dict) -> Result<(), &'static str> {
    println!("Received response: {}", Bencode::Dict(msg));
    Ok(())
}

fn krpc_on_error(msg: Dict) -> Result<(), &'static str> {
    println!("Received error response: {}", Bencode::Dict(msg));
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn random_node_id() -> NodeId {
        let mut node_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut node_id);

        node_id
    }

    fn build_contact() -> NodeContactInfo {
        NodeContactInfo {
            id: random_node_id(),
            address: "127.0.0.1:6881".parse().unwrap(),
        }
    }

    fn build_entry() -> RoutingEntry {
        RoutingEntry::new(build_contact())
    }

    fn build_contacts(count: usize) -> Vec<NodeContactInfo> {
        let mut result = Vec::with_capacity(count);

        for _ in 0..count {
            result.push(build_contact());
        }

        result
    }

    fn build_entries(count: usize) -> Vec<RoutingEntry> {
        let mut result = Vec::with_capacity(count);

        for _ in 0..count {
            result.push(build_entry());
        }

        result
    }

    fn bucket_contains(bucket: &Bucket, node: &NodeContactInfo) -> bool {
        let mut res = false;

        for i in 0..bucket.entries.len() {
            if &bucket.entries[i].node == node {
                res = true;
            }
        }

        res
    }

    #[test]
    fn node_id_common_prefix_length() {
        let id1 = [0u8; 20];
        let id2 = [0u8; 20];
        assert_eq!(160, node_id::common_prefix_length(&id1, &id2));

        let mut id2 = [255u8; 20];
        assert_eq!(0, node_id::common_prefix_length(&id1, &id2));

        id2[0] = 0;
        id2[1] = 0b00100000;
        assert_eq!(10, node_id::common_prefix_length(&id1, &id2));
    }

    #[test]
    fn find_node() {
        let mut rt = RoutingTable::new(random_node_id());
        let entry = build_entry();
        rt.buckets[0].entries.push(entry.clone());

        assert_eq!(&entry.node, rt.find_node(&entry.node.id).unwrap());
        assert!(rt.find_node(&build_contact().id).is_none());
    }

    #[test]
    fn find_closest_when_prefix_bucket_is_full() {
        let mut nodes = build_contacts(8);
        // The actual ids don't really matter; we're setting them manually here so
        // we can be sure we can search for one that is _not_ equal to any of these
        nodes[0].id[0] = 0b00000001;
        nodes[1].id[0] = 0b00000010;
        nodes[2].id[0] = 0b00000100;
        nodes[3].id[0] = 0b00001000;
        nodes[4].id[0] = 0b00010000;
        nodes[5].id[0] = 0b00100000;
        nodes[6].id[0] = 0b01000000;
        nodes[7].id[0] = 0b10000000;

        let entries = nodes
            .iter()
            .map(|contact| RoutingEntry {
                node: contact.clone(),
                last_query: None,
                last_response: None,
            })
            .collect();

        let mut rt = RoutingTable::new(random_node_id());
        rt.buckets[0].entries.append_vec(entries);

        let mut needle = random_node_id();
        needle[0] = 0b11111111;
        let closest_contacts = rt.find_closest(&needle);
        assert!(closest_contacts.as_slice().contains(&&nodes[0]));
        assert!(closest_contacts.as_slice().contains(&&nodes[1]));
        assert!(closest_contacts.as_slice().contains(&&nodes[2]));
        assert!(closest_contacts.as_slice().contains(&&nodes[3]));
        assert!(closest_contacts.as_slice().contains(&&nodes[4]));
        assert!(closest_contacts.as_slice().contains(&&nodes[5]));
        assert!(closest_contacts.as_slice().contains(&&nodes[6]));
        assert!(closest_contacts.as_slice().contains(&&nodes[7]));
    }

    #[test]
    fn find_closest_spanning_multiple_buckets() {
        let mut nodes = build_contacts(8);
        // The actual ids don't really matter; we're setting them manually here so
        // we can be sure we can search for one that is _not_ equal to any of these
        nodes[0].id[0] = 0b00000001;
        nodes[1].id[0] = 0b00000010;
        nodes[2].id[0] = 0b00000100;
        nodes[3].id[0] = 0b00001000;
        nodes[4].id[0] = 0b00010000;
        nodes[5].id[0] = 0b00100000;
        nodes[6].id[0] = 0b01000000;
        nodes[7].id[0] = 0b10000000;

        let mut entries: Vec<RoutingEntry> = nodes
            .iter()
            .map(|contact| RoutingEntry {
                node: contact.clone(),
                last_query: None,
                last_response: None,
            })
            .collect();

        let mut reference_id = random_node_id();
        let mut needle = random_node_id();
        reference_id[0] = 0b00001001;
        needle[0] = 0b00001011;

        let mut rt = RoutingTable::new(reference_id);
        rt.buckets[0].bounds = 0..2;
        rt.buckets.push_back(Bucket::new(2..5));
        rt.buckets.push_back(Bucket::new(5..100)); // this is the bucket the needle falls into (6 common prefix bits)
        rt.buckets.push_back(Bucket::new(100..160));

        // This dumps entries randomly into buckets. The common prefix length a bucket
        // represents, will most likely NOT be the correct one for the nodes.
        // In this case however, we don't care about the actual content of the buckets,
        // we just want to distribute nodes a little.
        rt.buckets[0].entries.push(entries.remove(0));
        rt.buckets[0].entries.push(entries.remove(0));
        rt.buckets[1].entries.push(entries.remove(0));
        rt.buckets[2].entries.push(entries.remove(0));
        rt.buckets[2].entries.push(entries.remove(0));
        rt.buckets[3].entries.push(entries.remove(0));
        rt.buckets[3].entries.push(entries.remove(0));
        rt.buckets[3].entries.push(entries.remove(0));

        let closest_contacts = rt.find_closest(&needle);
        assert!(closest_contacts.as_slice().contains(&&nodes[0]));
        assert!(closest_contacts.as_slice().contains(&&nodes[1]));
        assert!(closest_contacts.as_slice().contains(&&nodes[2]));
        assert!(closest_contacts.as_slice().contains(&&nodes[3]));
        assert!(closest_contacts.as_slice().contains(&&nodes[4]));
        assert!(closest_contacts.as_slice().contains(&&nodes[5]));
        assert!(closest_contacts.as_slice().contains(&&nodes[6]));
        assert!(closest_contacts.as_slice().contains(&&nodes[7]));
    }

    #[test]
    fn add_and_update_single_node() {
        let mut reference_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);

        let node = build_contact();
        let mut rt = RoutingTable::new(reference_id);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(0, rt.buckets[0].entries.len());

        rt.update(node.clone(), SeenIn::Referral);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(1, rt.buckets[0].entries.len());
        assert_eq!(node, rt.buckets[0].entries[0].node);
        assert!(rt.buckets[0].entries[0].last_query.is_none());
        assert!(rt.buckets[0].entries[0].last_response.is_none());

        rt.update(node.clone(), SeenIn::Query);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(1, rt.buckets[0].entries.len());
        assert_eq!(node, rt.buckets[0].entries[0].node);
        assert!(rt.buckets[0].entries[0].last_query.is_some());
        assert!(rt.buckets[0].entries[0].last_response.is_none());

        rt.update(node.clone(), SeenIn::Response);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(1, rt.buckets[0].entries.len());
        assert_eq!(node, rt.buckets[0].entries[0].node);
        assert!(rt.buckets[0].entries[0].last_query.is_some());
        assert!(rt.buckets[0].entries[0].last_response.is_some());
    }

    #[test]
    fn add_to_non_full_bucket() {
        let mut reference_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);

        let mut rt = RoutingTable::new(reference_id);
        // Leave one empty slot in bucket
        rt.buckets[0].entries.append_vec(build_entries(7));

        assert_eq!(1, rt.buckets.len());
        assert_eq!(7, rt.buckets[0].entries.len());

        rt.update(build_contact(), SeenIn::Referral);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(8, rt.buckets[0].entries.len());
    }

    #[test]
    fn replace_bad_node() {
        let mut reference_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);

        let mut rt = RoutingTable::new(reference_id);
        // Fill bucket
        for _ in 1..=8 {
            rt.buckets[0].entries.push(RoutingEntry {
                node: build_contact(),
                // Make sure this is a "good" entry
                last_query: Some(Instant::now()),
                last_response: Some(Instant::now()),
            });
        }
        // Make one node a "bad" one
        let bad_node = rt.buckets[0].entries[3].node.clone();
        rt.buckets[0].entries[3]
            .last_query
            .replace(Instant::now() - Duration::from_secs(60 * 23));
        rt.buckets[0].entries[3]
            .last_response
            .replace(Instant::now() - Duration::from_secs(60 * 23));

        assert_eq!(1, rt.buckets.len());
        assert_eq!(8, rt.buckets[0].entries.len());
        assert!(rt.buckets[0]
            .entries
            .iter()
            .any(|entry| entry.rating() == NodeRating::Bad));

        let new_node = build_contact();
        rt.update(new_node.clone(), SeenIn::Referral);

        assert_eq!(1, rt.buckets.len());
        assert_eq!(8, rt.buckets[0].entries.len());
        assert!(!bucket_contains(&rt.buckets[0], &bad_node));
        assert!(bucket_contains(&rt.buckets[0], &new_node));
    }

    #[test]
    fn split_bucket() {
        let mut reference_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);
        reference_id[0] = 0b11111111;
        let mut nodes = build_contacts(8);

        /* nodes near the reference */
        nodes[0].id[0] = 0b10000000;
        nodes[1].id[0] = 0b11000000;
        nodes[2].id[0] = 0b11100000;
        nodes[3].id[0] = 0b11110000;

        /* nodes further from the reference */
        nodes[4].id[0] = 0b00000000;
        nodes[5].id[0] = 0b01000000;
        nodes[6].id[0] = 0b01100000;
        nodes[7].id[0] = 0b01110000;

        let entries = nodes
            .iter()
            .map(|contact| RoutingEntry {
                node: contact.clone(),
                last_query: None,
                last_response: None,
            })
            .collect();

        let mut rt = RoutingTable::new(reference_id);
        rt.buckets[0].entries.append_vec(entries);

        let mut new_node = build_contact();
        // whether this node is near to or far from the reference doesn't matter,
        // we'll just pick one to make for a deterministic test
        new_node.id[0] = 0x00;

        rt.update(new_node.clone(), SeenIn::Referral);
        assert_eq!(2, rt.buckets.len());
        assert_eq!(0..1, rt.buckets[0].bounds);
        assert_eq!(1..160, rt.buckets[1].bounds);

        assert!(bucket_contains(&rt.buckets[0], &nodes[4]));
        assert!(bucket_contains(&rt.buckets[0], &nodes[5]));
        assert!(bucket_contains(&rt.buckets[0], &nodes[6]));
        assert!(bucket_contains(&rt.buckets[0], &nodes[7]));
        assert!(bucket_contains(&rt.buckets[0], &new_node));

        assert!(bucket_contains(&rt.buckets[1], &nodes[0]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[1]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[2]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[3]));
    }

    #[test]
    fn split_bucket_with_the_new_node_alone_in_one_half() {
        let mut reference_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);
        reference_id[0] = 0b11111111;
        let mut nodes = build_contacts(8);

        /* nodes near the reference */
        nodes[0].id[0] = 0b10000000;
        nodes[1].id[0] = 0b11000000;
        nodes[2].id[0] = 0b11100000;
        nodes[3].id[0] = 0b11110000;
        nodes[4].id[0] = 0b11111000;
        nodes[5].id[0] = 0b11111100;
        nodes[6].id[0] = 0b11111110;
        nodes[7].id[0] = 0b10000001;

        let entries = nodes
            .iter()
            .map(|contact| RoutingEntry {
                node: contact.clone(),
                last_query: None,
                last_response: None,
            })
            .collect();

        let mut rt = RoutingTable::new(reference_id);
        rt.buckets[0].entries.append_vec(entries);

        let mut new_node = build_contact();
        // new node is far from reference and will be alone in the new far bucket
        new_node.id[0] = 0x00;

        rt.update(new_node.clone(), SeenIn::Referral);
        assert_eq!(2, rt.buckets.len());
        assert_eq!(0..1, rt.buckets[0].bounds);
        assert_eq!(1..160, rt.buckets[1].bounds);

        assert!(bucket_contains(&rt.buckets[0], &new_node));

        assert!(bucket_contains(&rt.buckets[1], &nodes[0]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[1]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[2]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[3]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[4]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[5]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[6]));
        assert!(bucket_contains(&rt.buckets[1], &nodes[7]));
    }

    #[test]
    fn discard_node() {
        let reference_id = [0u8; 20];

        let mut rt = RoutingTable::new(reference_id);
        rt.buckets[0].bounds = 0..1;
        rt.buckets.push_back(Bucket::new(1..160));
        // Fill bucket
        for _ in 1..=8 {
            let mut node = build_contact();
            node.id[0] = 0xff;

            rt.buckets[0].entries.push(RoutingEntry {
                node,
                // Make sure this is a "good" entry
                last_query: Some(Instant::now()),
                last_response: Some(Instant::now()),
            });
        }

        assert_eq!(2, rt.buckets.len());
        assert_eq!(8, rt.buckets[0].entries.len());

        let mut new_node = build_contact();
        new_node.id[0] = 0xff;
        rt.update(new_node.clone(), SeenIn::Referral);

        assert_eq!(2, rt.buckets.len());
        assert_eq!(8, rt.buckets[0].entries.len());
        assert!(!bucket_contains(&rt.buckets[0], &new_node));
        assert!(!bucket_contains(&rt.buckets[1], &new_node));
    }

    #[test]
    fn entry_rating() {
        let mut entry = RoutingEntry::new(build_contact());
        assert_eq!(NodeRating::Questionable, entry.rating());

        entry.last_response = Some(Instant::now() - Duration::from_secs(60 * 10));
        assert_eq!(NodeRating::Good, entry.rating());

        entry.last_response = Some(Instant::now() - Duration::from_secs(60 * 20));
        assert_eq!(NodeRating::Bad, entry.rating());

        entry.last_query = Some(Instant::now() - Duration::from_secs(60 * 7));
        assert_eq!(NodeRating::Good, entry.rating());

        entry.last_query = Some(Instant::now() - Duration::from_secs(60 * 16));
        assert_eq!(NodeRating::Bad, entry.rating());

        // TODO: capture how often we tried to ping a node and mark it questionable
        //       before it goes bad.
    }
}
