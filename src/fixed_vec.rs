use std::mem::MaybeUninit;
use std::ops::{Index, IndexMut};

const BUCKET_SIZE: usize = 8;

pub struct FixedVec<T> {
    // TODO: For now this uses the fixed bep-005 bucket size of 8. It would be nice to
    //       generalize this once const generics are stable.
    data: [MaybeUninit<T>; BUCKET_SIZE],
    length: usize,
}

impl<T> FixedVec<T> {
    pub fn new() -> FixedVec<T> {
        FixedVec {
            data: unsafe { MaybeUninit::uninit().assume_init() },
            length: 0,
        }
    }

    fn empty_slots(&self) -> usize {
        BUCKET_SIZE - self.length
    }

    pub fn is_full(&self) -> bool {
        self.length == BUCKET_SIZE
    }

    pub fn is_not_full(&self) -> bool {
        self.length < BUCKET_SIZE
    }

    pub fn push(&mut self, item: T) {
        assert!(self.is_not_full());
        self.data[self.length] = MaybeUninit::new(item);
        self.length += 1;
    }

    pub fn len(&self) -> usize {
        self.length
    }

    pub fn as_slice(&self) -> &[T] {
        unsafe { std::slice::from_raw_parts(self.data[0].as_ptr(), self.length) }
    }

    pub fn iter(&self) -> std::slice::Iter<'_, T> {
        self.as_slice().iter()
    }

    pub fn swap_remove(&mut self, index: usize) -> T {
        assert!(index < self.length);
        let last_index = self.length - 1;
        unsafe {
            self.length -= 1;
            // If we're removing an element other than the last one, swap the last for it
            if index < last_index {
                std::ptr::replace(
                    self.data[index].as_mut_ptr(),
                    self.data[last_index].as_ptr().read(),
                )
            } else {
                self.data[index].as_ptr().read()
            }
        }
    }

    // TODO: append metods can probably be unified
    pub fn append(&mut self, other: FixedVec<T>) {
        assert!(self.empty_slots() >= other.len());
        unsafe {
            std::ptr::copy_nonoverlapping(other.data.as_ptr(), self.data.as_mut_ptr(), other.len());

            self.length += other.len()
        }
    }

    pub fn append_vec(&mut self, new_items: Vec<T>) {
        assert!(self.empty_slots() >= new_items.len());
        unsafe {
            std::ptr::copy_nonoverlapping(
                new_items.as_ptr(),
                self.data[self.length].as_mut_ptr(),
                new_items.len(),
            );

            self.length += new_items.len()
        }
    }
}

impl<T> Index<usize> for FixedVec<T> {
    type Output = T;

    fn index(&self, index: usize) -> &Self::Output {
        assert!(index < self.length);
        unsafe { &*self.data[index].as_ptr() }
    }
}

impl<T> IndexMut<usize> for FixedVec<T> {
    fn index_mut(&mut self, index: usize) -> &mut Self::Output {
        assert!(index < self.length);
        unsafe { &mut *self.data[index].as_mut_ptr() }
    }
}

impl<T> Drop for FixedVec<T> {
    fn drop(&mut self) {
        for item in &mut self.data[0..self.length] {
            unsafe {
                std::ptr::drop_in_place(item.as_mut_ptr());
            }
        }
    }
}

impl<T: std::fmt::Debug> std::fmt::Debug for FixedVec<T> {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        std::fmt::Debug::fmt(self.as_slice(), f)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn push() {
        let mut fvec = FixedVec::new();

        fvec.push("a");
        fvec.push("b");

        assert_eq!("a", fvec[0]);
        assert_eq!("b", fvec[1]);

        fvec.push("c");

        assert_eq!("a", fvec[0]);
        assert_eq!("b", fvec[1]);
        assert_eq!("c", fvec[2]);
    }

    #[test]
    fn append() {
        let mut other = FixedVec::new();
        other.push("a");
        other.push("b");
        other.push("c");

        let mut fvec = FixedVec::new();
        fvec.append(other);

        assert_eq!("a", fvec[0]);
        assert_eq!("b", fvec[1]);
        assert_eq!("c", fvec[2]);
    }

    #[test]
    fn append_vec() {
        let mut fvec = FixedVec::new();

        let vec = vec!["a", "b", "c"];
        fvec.append_vec(vec);

        assert_eq!("a", fvec[0]);
        assert_eq!("b", fvec[1]);
        assert_eq!("c", fvec[2]);
    }

    #[test]
    fn len_and_full() {
        let mut fvec = FixedVec::new();
        assert_eq!(0, fvec.len());
        assert_eq!(8, fvec.empty_slots());
        assert!(fvec.is_not_full());

        fvec.append_vec(vec!["a", "b"]);

        assert_eq!(2, fvec.len());
        assert_eq!(6, fvec.empty_slots());
        assert!(fvec.is_not_full());

        fvec.append_vec(vec!["c", "d", "e", "f", "g", "h"]);

        assert_eq!(8, fvec.len());
        assert_eq!(0, fvec.empty_slots());
        assert!(fvec.is_full());
    }

    #[test]
    fn iter() {
        let mut fvec = FixedVec::new();

        fvec.push("a");
        fvec.push("b");
        fvec.push("c");

        let vec: Vec<_> = fvec.iter().collect();

        assert_eq!(3, vec.len());
        assert_eq!(&"a", vec[0]);
        assert_eq!(&"b", vec[1]);
        assert_eq!(&"c", vec[2]);
    }

    #[test]
    fn swap_remove() {
        let mut fvec = FixedVec::new();

        fvec.push("a");
        fvec.push("b");
        fvec.push("c");

        fvec.swap_remove(0);
        assert_eq!(2, fvec.len());
        assert_eq!("c", fvec[0]);
        assert_eq!("b", fvec[1]);

        fvec.swap_remove(1);
        assert_eq!(1, fvec.len());
        assert_eq!("c", fvec[0]);
    }
}
