#!/bin/bash
go build
mkdir sfz2n64-1.0/usr/local/bin -p
cp sfz2n64 sfz2n64-1.0/usr/local/bin
mkdir sfz2n64-1.0/DEBIAN -p
cp control sfz2n64-1.0/DEBIAN
dpkg-deb --build sfz2n64-1.0