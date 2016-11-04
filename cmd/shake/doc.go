/*

Command shake computes the SHAKE variable-output-length hash functions
defined by FIPS-202.
It reads its entire input,
then produces output until it hits EOF.

Usage:

    shake [-n bits]

Flag -n specifies which function to compute, SHAKE128 or SHAKE256.
Bits can be 128 or 256. The default is 256.

Examples

Show 10 bytes of the hex-encoded SHAKE-256 digest of the string "hello":

    printf hello | shake | head -c 10 | hex

Obtain 2MB of the SHAKE-128 digest of the string "hello" in Go:

    cmd := exec.Command("shake", "-n", "128")
    cmd.Stdin = strings.NewReader("hello")
    cmd.Start()
    hash := make([]byte, 2e6)
    _, err := ioutil.ReadFull(cmd.Stdout.Read, hash)

*/
package main
