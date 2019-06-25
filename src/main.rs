mod bencode;

use bencode::{Bencode, Dict};
use rand::prelude::*;
use std::collections::BTreeMap;
use std::io;
use std::io::prelude::*;
use std::net::SocketAddrV4;
use std::net::UdpSocket;
use std::str;
use std::time::SystemTime;

#[derive(Debug)]
struct MyInfo {
    client_version: Option<String>,
    id: NodeId,
    port: u16,
}

const BOOTSTRAP_NODE: &str = "78.139.1.110:6881";

fn main() {
    let mut dht_node_id = vec![0; 20];
    rand::thread_rng().fill_bytes(&mut dht_node_id);
    let my_node_id = NodeId(dht_node_id);

    let my_info = MyInfo {
        client_version: None,
        id: my_node_id,
        port: 6881,
    };

    let socket = UdpSocket::bind("0.0.0.0:12345").unwrap();
    socket.set_nonblocking(true).unwrap();

    let mut buf = vec![0; 1024];

    let mut line = String::new();
    loop {
        print!("action: ");
        io::stdout().flush().unwrap();
        io::stdin().read_line(&mut line).unwrap();

        match line.trim().as_ref() {
            "p" => {
                println!("pinging...");
                socket.send_to(&build_ping(&my_info).encode(), BOOTSTRAP_NODE);
            }
            "f" => {
                println!("sending find_node...");
                socket.send_to(
                    &build_find_node_request(&my_info.id, &my_info).encode(),
                    BOOTSTRAP_NODE,
                );
            }
            "i" => {
                println!("checking inbox...");
                match socket.recv(&mut buf) {
                    Ok(num) => krpc_handle_noerr(Bencode::decode(&buf[..num]).unwrap()),
                    Err(_) => println!("No messages"),
                }
            }
            _ => println!("Invalid action: {}", line),
        }

        line.clear();
    }
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

#[derive(Debug, Clone)]
struct NodeId(Vec<u8>);

impl NodeId {
    fn from_vec(v: Vec<u8>) -> Result<NodeId, &'static str> {
        if v.len() != 20 {
            return Err("Invalid node id - must be 20 bytes long");
        }

        Ok(NodeId(v))
    }

    fn multiple_from_vec(v: Vec<u8>) -> Result<Vec<NodeId>, &'static str> {
        if v.len() % 20 != 0 {
            return Err("Invalid node ids - must be a multiple of 20 bytes");
        }

        Ok(v.chunks_exact(20).map(|c| NodeId(c.to_owned())).collect())
    }

    fn distance_to(&self, other: &NodeId) -> [u8; 20] {
        let mut result = [0u8; 20];

        for i in 0..20 {
            result[i] = self.0[i] ^ self.0[i];
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
    build_request(b"ping", BTreeMap::new(), my_info)
}

fn build_find_node_request(target: &NodeId, my_info: &MyInfo) -> Bencode {
    let mut args = BTreeMap::new();
    args.insert(b"target".to_vec(), Bencode::Bytes(target.0.to_owned()));

    build_request(b"find_node", args, my_info)
}

fn build_request(query_type: &[u8], mut args: Dict, my_info: &MyInfo) -> Bencode {
    let mut dict = BTreeMap::new();
    dict.insert(b"t".to_vec(), Bencode::Bytes(query_type.to_owned()));
    dict.insert(b"y".to_vec(), Bencode::Bytes(b"q".to_vec()));
    dict.insert(b"q".to_vec(), Bencode::Bytes(query_type.to_owned()));

    if let Some(ref cv) = my_info.client_version {
        dict.insert(b"v".to_vec(), Bencode::Bytes(cv.as_bytes().to_owned()));
    }

    args.insert(b"id".to_vec(), Bencode::Bytes(my_info.id.0.to_owned()));
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
            Some(Bencode::Bytes(v)) => NodeId::from_vec(v)?,
            _ => return Err("No or invalid responder id"),
        };

        let node_ids = match return_values.remove::<[u8]>(b"nodes") {
            Some(Bencode::Bytes(v)) => NodeId::multiple_from_vec(v)?,
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
        _ => return Err("KRPC message does not contain key y or invalid value type")
    };

    match &y[..] {
        b"r" => krpc_on_return(msg),
        b"e" => krpc_on_error(msg),
        _ => Err("Unhandled KRPC message type")
    }
}

fn krpc_on_call(msg: Dict) -> Result<(), &'static str> {
    Err("call not implemented")
}

fn krpc_on_return(msg: Dict) -> Result<(), &'static str> {
    println!("Received response: {}", Bencode::Dict(msg));
    Ok(())
}

fn krpc_on_error(msg: Dict) -> Result<(), &'static str> {
    println!("Received error reponse: {}", Bencode::Dict(msg));
    Ok(())
}
