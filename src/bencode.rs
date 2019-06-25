use regex::bytes::Regex;
use std::borrow::Cow;
use std::collections::BTreeMap;
use std::str;

struct Decoder<'a> {
    data: &'a [u8],
}

impl<'a> Decoder<'a> {
    fn parse(data: &[u8]) -> Result<Bencode, DecoderError> {
        Decoder { data }.parse_value()
    }

    fn remaining(&self) -> usize {
        self.data.len()
    }

    fn peek_byte(&self) -> Result<u8, DecoderError> {
        if self.remaining() == 0 {
            Err(DecoderError::EndOfStream)
        } else {
            Ok(self.data[0])
        }
    }

    fn advance(&mut self, count: usize) {
        let new_start = std::cmp::min(count, self.remaining());
        self.data = &self.data[new_start..];
    }

    fn take_byte(&mut self) -> Result<u8, DecoderError> {
        let result = self.peek_byte();
        self.advance(1);
        result
    }

    fn parse_value(&mut self) -> Result<Bencode, DecoderError> {
        let next_byte = self.peek_byte()?;

        match next_byte {
            b'i' => self.parse_int(),
            b'0'...b'9' => self.parse_bytes(),
            b'l' => self.parse_list(),
            b'd' => self.parse_dict(),
            _ => Err(DecoderError::UnexpectedStartOfValue),
        }
    }

    fn parse_int(&mut self) -> Result<Bencode, DecoderError> {
        let raw_bytes = Regex::new(r"^i(-?\d+)e")
            .unwrap()
            .captures(self.data)
            .ok_or(DecoderError::ExpectedInteger)?
            .get(1)
            .ok_or(DecoderError::ExpectedInteger)?
            .as_bytes();

        self.advance(raw_bytes.len() + 2);

        let integer = str::from_utf8(raw_bytes).unwrap().parse().unwrap();
        Ok(Bencode::Integer(integer))
    }

    fn parse_bytes(&mut self) -> Result<Bencode, DecoderError> {
        let raw_string_size = Regex::new(r"^(\d+):")
            .unwrap()
            .captures(self.data)
            .ok_or(DecoderError::ExpectedStringStart)?
            .get(1)
            .ok_or(DecoderError::ExpectedStringStart)?
            .as_bytes();

        self.advance(raw_string_size.len() + 1);

        let string_size = str::from_utf8(raw_string_size).unwrap().parse().unwrap();

        if self.remaining() < string_size {
            return Err(DecoderError::InvalidStringSize);
        }

        let bytes = self.data[..string_size].to_owned();

        self.advance(string_size);

        Ok(Bencode::Bytes(bytes))
    }

    fn parse_list(&mut self) -> Result<Bencode, DecoderError> {
        if self.take_byte()? != b'l' {
            return Err(DecoderError::ExpectedListStart);
        }

        let mut result = Vec::new();

        loop {
            let next_byte = self.peek_byte()?;

            match next_byte {
                b'e' => {
                    self.take_byte()?;
                    break;
                }
                _ => result.push(self.parse_value()?),
            }
        }

        Ok(Bencode::List(result))
    }

    fn parse_dict(&mut self) -> Result<Bencode, DecoderError> {
        if self.take_byte()? != b'd' {
            return Err(DecoderError::ExpectedDictStart);
        }

        let mut result = BTreeMap::new();

        loop {
            let next_byte = self.peek_byte()?;

            match next_byte {
                b'e' => {
                    self.take_byte()?;
                    break;
                }
                _ => {
                    let key = match self.parse_bytes()? {
                        Bencode::Bytes(bytes) => bytes,
                        _ => return Err(DecoderError::ExpectedStringKey),
                    };
                    let value = self.parse_value()?;

                    result.insert(key, value);
                }
            }
        }

        Ok(Bencode::Dict(result))
    }
}

#[derive(Debug)]
pub enum DecoderError {
    EndOfStream,
    ExpectedInteger,
    ExpectedStringStart,
    ExpectedListStart,
    ExpectedDictStart,
    ExpectedStringKey,
    InvalidStringSize,
    UnexpectedStartOfValue,
}

pub type Dict = BTreeMap<Vec<u8>, Bencode>;

#[derive(Debug, PartialEq)]
pub enum Bencode {
    Bytes(Vec<u8>),
    Integer(isize),
    List(Vec<Bencode>),
    Dict(Dict),
}

impl Bencode {
    pub fn decode(input: &[u8]) -> Result<Bencode, DecoderError> {
        Decoder::parse(input)
    }

    pub fn encode(self) -> Vec<u8> {
        match self {
            Bencode::Bytes(mut v) => {
                let mut r = Vec::new();
                r.append(&mut v.len().to_string().into_bytes());
                r.push(b':');
                r.append(&mut v);
                r
            }
            Bencode::Integer(v) => {
                let mut r = Vec::new();
                r.push(b'i');
                r.append(&mut v.to_string().into_bytes());
                r.push(b'e');
                r
            }
            Bencode::List(values) => {
                let mut r = Vec::new();
                r.push(b'l');
                for v in values {
                    r.append(&mut v.encode());
                }
                r.push(b'e');
                r
            }
            Bencode::Dict(map) => {
                let mut r = Vec::new();
                r.push(b'd');
                for (k, v) in map {
                    r.append(&mut Bencode::Bytes(k).encode());
                    r.append(&mut v.encode());
                }
                r.push(b'e');
                r
            }
        }
    }
}

fn fmt_bytes(b: &[u8]) -> Cow<str> {
    match str::from_utf8(b) {
        Ok(v) => Cow::Borrowed(v),
        Err(_) => Cow::Owned(format!("{:02X?}", b)),
    }
}

impl std::fmt::Display for Bencode {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Bencode::Bytes(b) => write!(f, "{}", fmt_bytes(b)),
            Bencode::Integer(i) => write!(f, "{}", i),
            Bencode::List(l) => {
                let l_str = l
                    .iter()
                    .map(|v| format!("{}", v))
                    .collect::<Vec<String>>()
                    .join(", ");

                write!(f, "[{}]", l_str)
            }
            Bencode::Dict(d) => {
                let d_str = d
                    .iter()
                    .map(|(k, v)| format!("{} => {}", fmt_bytes(k), format!("{}", v)))
                    .collect::<Vec<String>>()
                    .join(", ");

                write!(f, "{{{}}}", d_str)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_int() {
        assert_eq!(
            Bencode::Integer(-123),
            Decoder { data: b"i-123e" }.parse_int().unwrap()
        )
    }

    #[test]
    fn test_parse_bytes() {
        assert_eq!(
            Bencode::Bytes(b"abc".to_vec()),
            Decoder { data: b"3:abcxyz" }.parse_bytes().unwrap()
        )
    }

    #[test]
    fn test_parse_list() {
        assert_eq!(
            Bencode::List(vec![Bencode::Integer(4), Bencode::Bytes(b"qwe".to_vec())]),
            Decoder {
                data: b"li4e3:qwee"
            }
            .parse_list()
            .unwrap()
        )
    }

    #[test]
    fn test_parse_dict() {
        let mut map = BTreeMap::new();
        map.insert(b"one".to_vec(), Bencode::Bytes(b"hello".to_vec()));
        map.insert(b"two".to_vec(), Bencode::Integer(123));

        assert_eq!(
            Bencode::Dict(map),
            Decoder {
                data: b"d3:one5:hello3:twoi123ee"
            }
            .parse_dict()
            .unwrap()
        )
    }
}
