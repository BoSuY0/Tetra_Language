fn lookup_or(keys: &[i32], values: &[i32], key: i32, fallback: i32) -> i32 {
    let limit = keys.len().min(values.len());
    for i in 0..limit {
        if keys[i] == key {
            return values[i];
        }
    }
    fallback
}

fn main() {
    let n: i32 = 1024;
    let mut keys = vec![0_i32; n as usize];
    let mut values = vec![0_i32; n as usize];
    for i in 0..n {
        keys[i as usize] = i * 2 + 1;
        values[i as usize] = i + 7;
    }

    let mut checksum = 0_i32;
    for q in 0..n {
        checksum += lookup_or(&keys, &values, q * 2 + 1, 0);
    }
    println!("{}", checksum);
    std::process::exit(if checksum == 530944 { 0 } else { 1 });
}
