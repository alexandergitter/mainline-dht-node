mod bencode;

use bencode::{Bencode, Dict};
use cursive::view::*;
use cursive::views::*;
use cursive::Cursive;
use rand::prelude::*;
use std::collections::VecDeque;
use std::net::{SocketAddrV4, ToSocketAddrs, UdpSocket};
use std::str;
use std::sync::mpsc;
use std::thread;
use std::time::{Duration, SystemTime};

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
const BUCKET_SIZE: usize = 8;

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

#[derive(Debug)]
struct RoutingEntry {
    node: NodeContactInfo,
    last_response: Option<SystemTime>,
    last_query: Option<SystemTime>,
}

#[derive(Debug)]
struct Bucket {
    nodes: Vec<RoutingEntry>,
    last_changed: SystemTime,
    upper_prefix_len: u32,
}

#[derive(Debug)]
struct RoutingTable {
    data: VecDeque<Bucket>,
    reference_id: NodeId,
}

enum SeenIn {
    Request,
    Response,
    Referral,
}

impl RoutingTable {
    fn new(reference_id: NodeId) -> RoutingTable {
        // TODO: figure out the avg. number of nodes added on bootstrap and set initial size accordingly
        RoutingTable {
            data: VecDeque::new(),
            reference_id,
        }
    }

    fn find_neighbors(&self, node_id: NodeId) {}

    fn update(&mut self, node: NodeContactInfo, seenIn: SeenIn) {
        if self.data.is_empty() {
            self.data.push_back(Bucket {
                nodes: Vec::with_capacity(BUCKET_SIZE),
                last_changed: SystemTime::now(),
                upper_prefix_len: 160,
            })
        }

        let distance = node_id::distance(&self.reference_id, &node.id);
        let mut prefix_len = 0;

        for byte in &distance {
            prefix_len += byte.leading_zeros();

            if byte.leading_zeros() < 8 {
                break;
            }
        }

        let bucket = self
            .data
            .iter_mut()
            .find(|bucket| prefix_len < bucket.upper_prefix_len)
            .expect(&format!(
                "no bucket for prefix length {} - this should never happen",
                prefix_len
            ));

        bucket.nodes.push(RoutingEntry {
            node,
            last_query: None,
            last_response: None,
        });
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

        #[allow(clippy::needless_range_loop)]
        for i in 0..result.len() {
            result[i] = a[i] ^ b[i];
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
    println!("Received error reponse: {}", Bencode::Dict(msg));
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn add_and_update_single_node() {
        let mut reference_id = [0u8; 20];
        let mut node_id = [0u8; 20];
        rand::thread_rng().fill_bytes(&mut reference_id);
        rand::thread_rng().fill_bytes(&mut node_id);

        let node = NodeContactInfo {
            id: node_id,
            address: "127.0.0.1:6881".parse().unwrap(),
        };

        let mut rt = RoutingTable::new(reference_id);

        rt.update(node.clone(), SeenIn::Request);

        assert_eq!(1, rt.data.len());
        assert_eq!(1, rt.data[0].nodes.len());
        assert_eq!(node, rt.data[0].nodes[0].node);
        assert!(rt.data[0].nodes[0].last_query.is_none());
        assert!(rt.data[0].nodes[0].last_response.is_none());
    }
}
