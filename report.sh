#!/bin/bash

echo "==================================="
echo "===== EROFS Comparison Script ====="
echo "==================================="

echo "Creating Go/C reference images..."
../erofs-utils/mkfs/mkfs.erofs -d 4 c_reference.img test_data/
./mkfs.erofs -d 4 -o go_reference.img -i test_data/

echo "Hexdump images..."
hexdump -C -n 2048 c_reference.img > report/c_hexdump.txt
hexdump -C -n 2048 go_reference.img > report/go_hexdump.txt

echo "diff images..."
diff -u report/c_hexdump.txt report/go_hexdump.txt > report/diff.log

echo "Extraction superblock data..."
dd if=c_reference.img of=report/c_superblock.bin bs=1024 skip=1 count=1
dd if=go_reference.img of=report/go_superblock.bin bs=1024 skip=1 count=1
hexdump -C report/c_superblock.bin > report/c_superblock.txt
hexdump -C report/go_superblock.bin > report/go_superblock.txt

echo "Superblock data Comparision..."
diff -u report/c_superblock.txt report/go_superblock.txt > report/diff_superblock.log