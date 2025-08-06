// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"bufio"
	"bytes"
	"io"
	"math"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// Blob represents a Git object.
type Blob struct {
	ID SHA1

	gotSize bool
	size    int64
	name    string
	repo    *Repository
}

// DataAsync gets a ReadCloser for the contents of a blob without reading it all.
// Calling the Close function on the result will discard all unread output.
func (b *Blob) DataAsync() (io.ReadCloser, error) {
	blobClient, err := b.repo.BlobClient.GetBlob(b.repo.Ctx, &gitalypb.GetBlobRequest{Repository: b.repo.GitalyRepo, Oid: b.ID.String(), Limit: -1})
	if err != nil {
		return nil, err
	}

	resp := make([]byte, 0, 4096)
	canRead := true
	for canRead {
		blobResponse, _ := blobClient.Recv()
		if blobResponse == nil {
			canRead = false
		} else {
			resp = append(resp, blobResponse.Data...)
		}
	}

	return io.NopCloser(bytes.NewReader(resp)), nil
}

// Size returns the uncompressed size of the blob
func (b *Blob) Size() int64 {
	if b.gotSize {
		return b.size
	}

	blobClient, err := b.repo.BlobClient.GetBlob(b.repo.Ctx, &gitalypb.GetBlobRequest{Repository: b.repo.GitalyRepo, Oid: b.ID.String(), Limit: -1})
	if err != nil {
		return 0
	}

	blobResponse, err := blobClient.Recv()
	if err != nil {
		return 0
	}
	b.size = blobResponse.Size
	b.gotSize = true

	return b.size
}

type blobReader struct {
	rd     *bufio.Reader
	n      int64
	cancel func()
}

func (b *blobReader) Read(p []byte) (n int, err error) {
	if b.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > b.n {
		p = p[0:b.n]
	}
	n, err = b.rd.Read(p)
	b.n -= int64(n)
	return n, err
}

// Close implements io.Closer
func (b *blobReader) Close() error {
	defer b.cancel()
	if b.n > 0 {
		for b.n > math.MaxInt32 {
			n, err := b.rd.Discard(math.MaxInt32)
			b.n -= int64(n)
			if err != nil {
				return err
			}
			b.n -= math.MaxInt32
		}
		n, err := b.rd.Discard(int(b.n))
		b.n -= int64(n)
		if err != nil {
			return err
		}
	}
	if b.n == 0 {
		_, err := b.rd.Discard(1)
		b.n--
		return err
	}
	return nil
}
