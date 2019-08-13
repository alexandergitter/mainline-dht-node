use std::borrow::Cow;
use std::collections::BTreeMap;
use std::str;

struct Cursor<'a> {
    data: &'a [u8],
    position: usize,
}

impl<'a> Cursor<'a> {
    fn new(data: &[u8]) -> Cursor {
        Cursor { data, position: 0 }
    }

    fn remaining(&self) -> usize {
        self.data.len() - self.position
    }

    fn peek_byte(&self) -> Result<u8, DecoderError> {
        if self.remaining() == 0 {
            Err(DecoderError::EndOfStream)
        } else {
            Ok(self.data[self.position])
        }
    }

    fn advance(&mut self, count: usize) {
        let safe_offset = std::cmp::min(count, self.remaining());
        self.position += safe_offset;
    }

    fn take_byte(&mut self) -> Result<u8, DecoderError> {
        let result = self.peek_byte();
        self.advance(1);
        result
    }

    fn get_slice(&self) -> &[u8] {
        &self.data[self.position..]
    }
}

struct Decoder<'a> {
    cursor: Cursor<'a>,
}

impl<'a> Decoder<'a> {
    fn new(data: &[u8]) -> Decoder {
        Decoder {
            cursor: Cursor::new(data),
        }
    }

    fn parse_int(&mut self) -> Result<i64, DecoderError> {
        let is_negative = match self.cursor.peek_byte()? {
            b'-' => {
                self.cursor.advance(1);
                true
            }
            b'+' => {
                self.cursor.advance(1);
                false
            }
            _ => false,
        };

        if !(self.cursor.peek_byte()?.is_ascii_digit()) {
            return Err(DecoderError::ExpectedInteger);
        }

        let mut result = 0i64;

        while (self.cursor.remaining() > 0) && (self.cursor.peek_byte()?.is_ascii_digit()) {
            let current_digit = (self.cursor.take_byte()? - b'0') as i64;

            result = result
                .checked_mul(10)
                .ok_or(DecoderError::OversizedInteger)?;
            if is_negative {
                result = result
                    .checked_sub(current_digit)
                    .ok_or(DecoderError::OversizedInteger)?;
            } else {
                result = result
                    .checked_add(current_digit)
                    .ok_or(DecoderError::OversizedInteger)?;
            }
        }

        Ok(result)
    }

    fn decode_value(&mut self) -> Result<Bencode, DecoderError> {
        match self.cursor.peek_byte()? {
            b'i' => self.decode_int(),
            b'0'...b'9' => self.decode_bytestring(),
            b'l' => self.decode_list(),
            b'd' => self.decode_dict(),
            _ => Err(DecoderError::UnexpectedStartOfValue),
        }
    }

    fn decode_int(&mut self) -> Result<Bencode, DecoderError> {
        if self.cursor.take_byte()? != b'i' {
            return Err(DecoderError::ExpectedIntegerStart);
        }

        let integer = self.parse_int()?;

        if self.cursor.take_byte()? != b'e' {
            return Err(DecoderError::ExpectedIntegerEnd);
        }

        Ok(Bencode::Integer(integer))
    }

    fn decode_bytestring(&mut self) -> Result<Bencode, DecoderError> {
        let string_size = self.parse_int()? as usize;

        if self.cursor.take_byte()? != b':' {
            return Err(DecoderError::ExpectedStringStart);
        }

        if self.cursor.remaining() < string_size {
            return Err(DecoderError::InvalidStringSize);
        }

        let bytes = self.cursor.get_slice()[..string_size].to_owned();

        self.cursor.advance(string_size);

        Ok(Bencode::Bytes(bytes))
    }

    fn decode_list(&mut self) -> Result<Bencode, DecoderError> {
        if self.cursor.take_byte()? != b'l' {
            return Err(DecoderError::ExpectedListStart);
        }

        let mut result = Vec::new();

        loop {
            let next_byte = self.cursor.peek_byte()?;

            match next_byte {
                b'e' => {
                    self.cursor.take_byte()?;
                    break;
                }
                _ => result.push(self.decode_value()?),
            }
        }

        Ok(Bencode::List(result))
    }

    fn decode_dict(&mut self) -> Result<Bencode, DecoderError> {
        if self.cursor.take_byte()? != b'd' {
            return Err(DecoderError::ExpectedDictStart);
        }

        let mut result = BTreeMap::new();

        loop {
            let next_byte = self.cursor.peek_byte()?;

            match next_byte {
                b'e' => {
                    self.cursor.take_byte()?;
                    break;
                }
                _ => {
                    let key = match self.decode_bytestring()? {
                        Bencode::Bytes(bytes) => bytes,
                        // NOTE: This should never happen,because we are diverting errors
                        //       with the ? above. And decode_bytestring always returns a
                        //       Bencode::Bytes; unfortunately this cannot be expressed yet.
                        //       https://github.com/rust-lang/rfcs/pull/2593
                        other => unreachable!("unexpected bencode type: {}", other),
                    };
                    let value = self.decode_value()?;

                    result.insert(key, value);
                }
            }
        }

        Ok(Bencode::Dict(result))
    }
}

#[derive(Debug, PartialEq)]
pub enum DecoderError {
    EndOfStream,
    OversizedInteger,
    ExpectedInteger,
    ExpectedIntegerStart,
    ExpectedIntegerEnd,
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
    Integer(i64),
    List(Vec<Bencode>),
    Dict(Dict),
}

impl Bencode {
    pub fn decode(input: &[u8]) -> Result<Bencode, DecoderError> {
        Decoder::new(input).decode_value()
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
        assert_eq!(123, Decoder::new(b"123").parse_int().unwrap());
        assert_eq!(123, Decoder::new(b"+123").parse_int().unwrap());
        assert_eq!(-123, Decoder::new(b"-123").parse_int().unwrap());
        assert_eq!(
            DecoderError::ExpectedInteger,
            Decoder::new(b"-a").parse_int().unwrap_err()
        );
        assert_eq!(
            DecoderError::EndOfStream,
            Decoder::new(b"").parse_int().unwrap_err()
        );
        assert_eq!(
            DecoderError::OversizedInteger,
            Decoder::new(b"-100000000000000000000")
                .parse_int()
                .unwrap_err()
        );
    }

    #[test]
    fn test_decode_int() {
        assert_eq!(
            Bencode::Integer(-123),
            Decoder::new(b"i-123e").decode_int().unwrap()
        )
    }

    #[test]
    fn test_decode_bytestring() {
        assert_eq!(
            Bencode::Bytes(b"abc".to_vec()),
            Decoder::new(b"3:abcxyz").decode_bytestring().unwrap()
        )
    }

    #[test]
    fn test_decode_list() {
        assert_eq!(
            Bencode::List(vec![Bencode::Integer(4), Bencode::Bytes(b"qwe".to_vec())]),
            Decoder::new(b"li4e3:qwee").decode_list().unwrap()
        )
    }

    #[test]
    fn test_decode_dict() {
        let mut map = BTreeMap::new();
        map.insert(b"one".to_vec(), Bencode::Bytes(b"hello".to_vec()));
        map.insert(b"two".to_vec(), Bencode::Integer(123));

        assert_eq!(
            Bencode::Dict(map),
            Decoder::new(b"d3:one5:hello3:twoi123ee")
                .decode_dict()
                .unwrap()
        )
    }
}
