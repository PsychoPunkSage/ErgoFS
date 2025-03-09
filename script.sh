#!/bin/bash

set -e

echo "===== EROFS Testing Script ====="
rm -rf test_data test.img mount_point mkfs.erofs erofs-verify

# Create test directory and sample files
echo "Creating test data..."
mkdir -p test_data/subdir
echo "Hello, EROFS!" > test_data/hello.txt
echo "This is a test file in a subdirectory" > test_data/subdir/test.txt
dd if=/dev/urandom of=test_data/random.bin bs=1M count=1 status=progress 2>/dev/null
ln -s hello.txt test_data/symlink

# Build tools
echo "Building tools..."
go build -o mkfs.erofs ./cmd/mkfs
# go build -o erofs-verify ./cmd/verify

# Create EROFS image
echo "Creating EROFS image..."
# ./mkfs.erofs -d 4 -o test.img -i test_data
./mkfs.erofs -d 4 -o test.img -i /home/psychopunk_sage/dev/OpenSource/Unikraft/erofs-utils/test_data

# Verify the image
# echo "Verifying image before fix..."
# ./erofs-verify test.img

# Verify again after fixing
# echo "Verifying image after fix..."
# ./erofs-verify test.img

# Create mount point
echo "Creating mount point..."
mkdir -p mount_point

# Try to mount
echo "Attempting to mount the image..."
if sudo mount -t erofs -o ro test.img mount_point; then
    echo "Mount successful!"
    
    # List the contents
    echo "Listing mount point contents:"
    ls -la mount_point
    
    # Verify content
    echo "Verifying file content:"
    cat mount_point/hello.txt
    
    # Unmount
    echo "Unmounting..."
    sudo umount mount_point
else
    echo "Mount failed. Checking kernel messages..."
    sudo dmesg | tail -20
fi

echo "===== Test Complete ====="