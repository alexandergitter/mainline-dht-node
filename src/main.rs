mod bencode;

use bencode::{Bencode, Dict};
use cursive::view::*;
use cursive::views::*;
use cursive::Cursive;
use rand::prelude::*;
use std::net::{SocketAddrV4, UdpSocket};
use std::str;
use std::thread;
use std::time::{Duration, Instant, SystemTime};

#[derive(Debug)]
struct MyInfo {
    client_version: Option<String>,
    id: NodeId,
    port: u16,
}

const BOOTSTRAP_NODE: &str = "78.139.1.110:6881";

fn main() {
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

    // Creates the cursive root - required for every application.
    let mut cs = Cursive::default();

    cs.set_user_data((my_info, socket));

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
                            let (my_info, _) = cs.user_data::<(MyInfo, UdpSocket)>().unwrap();

                            let mut args = Dict::new();
                            args.insert(b"id".to_vec(), Bencode::Bytes(my_info.id.to_vec()));
                            args.insert(b"target".to_vec(), Bencode::Bytes(my_info.id.to_vec()));

                            cs.pop_layer();
                            let res = krpc_sync("find_node", args, val, cs);
                        }),
                ),
        );
    });

    cs.add_global_callback('i', move |cs| {
        let (_, socket) = cs.user_data::<(MyInfo, UdpSocket)>().unwrap();
        match socket.recv(&mut buf) {
            Ok(num) => log(cs, Bencode::decode(&buf[..num]).unwrap()),
            Err(_) => log(cs, "No messages"),
        };
    });

    cs.run();
}

fn log<T: std::fmt::Display>(cs: &mut Cursive, msg: T) {
    cs.call_on_id("main_content", |view: &mut TextView| {
        let new_content = format!("\n===\n{}", msg);
        view.append(new_content)
    });
}

fn krpc_sync<A: std::net::ToSocketAddrs>(
    method: &str,
    args: Dict,
    to_addr: A,
    cs: &mut Cursive,
) -> Result<Bencode, KrpcError> {
    let mut transaction_id = [0u8; 2];
    rand::thread_rng().fill_bytes(&mut transaction_id);

    let mut dict = Dict::new();
    dict.insert(b"t".to_vec(), Bencode::Bytes(transaction_id.to_vec()));
    dict.insert(b"y".to_vec(), Bencode::Bytes(b"q".to_vec()));
    dict.insert(b"q".to_vec(), Bencode::Bytes(method.as_bytes().to_owned()));
    dict.insert(b"a".to_vec(), Bencode::Dict(args));

    let (my_info, _) = cs.user_data::<(MyInfo, UdpSocket)>().unwrap();

    if let Some(ref client_version) = my_info.client_version {
        dict.insert(
            b"v".to_vec(),
            Bencode::Bytes(client_version.as_bytes().to_owned()),
        );
    }

    let bencode = Bencode::Dict(dict);
    log(cs, format!("sending: {}", bencode));

    let (_, socket) = cs.user_data::<(MyInfo, UdpSocket)>().unwrap();

    let res = socket.send_to(&bencode.encode(), to_addr);
    if let Err(_) = res {
        log(cs, "error sending");
        return Err(KrpcError::SendError);
    }

    let mut buf = vec![0; 1024];
    let start_time = Instant::now();
    let (_, socket) = cs.user_data::<(MyInfo, UdpSocket)>().unwrap();

    while start_time.elapsed().as_secs() < 10 {
        match socket.recv(&mut buf) {
            Ok(num) => {
                let result = Bencode::decode(&buf[..num]).unwrap();
                log(cs, format!("received: {}", result));
                return Ok(result);
            }
            Err(_) => {}
        };

        thread::sleep(Duration::from_millis(50));
    }

    log(cs, "Waiting for answer timed out");
    Err(KrpcError::ResponseTimeout)
}

enum KrpcError {
    SendError,
    ResponseTimeout,
    MalformedResponse,
    InvalidResponse,
    ErrorResponse,
}

#[derive(Debug)]
struct DHTRoutingTable {
    nodes: Vec<DHTNode>,
}

#[derive(Debug)]
struct DHTNode {
    contact: NodeContactInfo,
    last_response: Option<SystemTime>,
    last_query: Option<SystemTime>,
}

type NodeId = [u8; 20];

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
struct NodeContactInfo {
    id: NodeId,
    address: SocketAddrV4,
}

fn build_ping(my_info: &MyInfo) -> Bencode {
    build_request(b"ping", Dict::new(), my_info)
}

fn build_find_node_request(target: &NodeId, my_info: &MyInfo) -> Bencode {
    let mut args = Dict::new();
    args.insert(b"target".to_vec(), Bencode::Bytes(target.to_vec()));

    build_request(b"find_node", args, my_info)
}

fn build_request(query_type: &[u8], mut args: Dict, my_info: &MyInfo) -> Bencode {
    let mut dict = Dict::new();
    dict.insert(b"t".to_vec(), Bencode::Bytes(query_type.to_owned()));
    dict.insert(b"y".to_vec(), Bencode::Bytes(b"q".to_vec()));
    dict.insert(b"q".to_vec(), Bencode::Bytes(query_type.to_owned()));

    if let Some(ref cv) = my_info.client_version {
        dict.insert(b"v".to_vec(), Bencode::Bytes(cv.as_bytes().to_owned()));
    }

    args.insert(b"id".to_vec(), Bencode::Bytes(my_info.id.to_vec()));
    dict.insert(b"a".to_vec(), Bencode::Dict(args));

    Bencode::Dict(dict)
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

fn krpc_handle_noerr(bencode: Bencode) {
    match krpc_handle(bencode) {
        Ok(_) => (),
        Err(what) => println!("Error in response handling: {}", what),
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
