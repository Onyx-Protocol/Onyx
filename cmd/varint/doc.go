/*

Command varint encodes a decimal number to or from varint.

Usage:

      varint

It reads from stdin when decoding, and takes a parameter when encoding.

Examples:

Obtain the decimal value of the hex-encoded varint 0101:

      printf 0101 | hex -d | varint

Obtain the unsigned varint value of the decimal number 1234:

      varint 1234 | hex

Note that encoding a varint without hex encoding it will often result in an
byte value that cannot be printed as ASCII.

*/
package main
