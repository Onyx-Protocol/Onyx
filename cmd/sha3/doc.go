/*

Command sha3 prints the binary SHA-3 digest of its input.
It computes the fixed-output-length hash functions defined by FIPS-202,
and produces exactly length/8 bytes of output.

Usage:

    sha3 [-n length]

Flag -n specifies which size output to compute.
Length can be 224, 256, 384, or 512.
The default is 256.
Note that the shorter lengths are not just a truncated
form of the longer ones. FIPS 202 specifies different
construction parameters for each output size,
so they compute different hash functions.

Examples

Show the hex-encoded SHA3-256 digest of the string "hello":

    printf hello | sha3 | hex

Obtain the 64-byte SHA3-512 digest of the string "hello" in Go:

    cmd := exec.Command("sha3", "-n", "512")
    cmd.Stdin = strings.NewReader("hello")
    hash, err := cmd.Output()

*/
package main
