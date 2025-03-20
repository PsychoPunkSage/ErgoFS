go build -o mkfs.erofs ./cmd/mkfs
./mkfs.erofs -d 4 output.img test_data/