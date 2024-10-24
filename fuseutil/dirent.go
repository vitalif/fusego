// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fuseutil

import (
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/internal/fusekernel"
)

type DirentType uint32

const (
	DT_Unknown   DirentType = 0
	DT_Socket    DirentType = syscall.DT_SOCK
	DT_Link      DirentType = syscall.DT_LNK
	DT_File      DirentType = syscall.DT_REG
	DT_Block     DirentType = syscall.DT_BLK
	DT_Directory DirentType = syscall.DT_DIR
	DT_Char      DirentType = syscall.DT_CHR
	DT_FIFO      DirentType = syscall.DT_FIFO
)

// A struct representing an entry within a directory file, describing a child.
// See notes on fuseops.ReadDirOp and on WriteDirent for details.
type Dirent struct {
	// The (opaque) offset within the directory file of the entry following this
	// one. See notes on fuseops.ReadDirOp.Offset for details.
	Offset fuseops.DirOffset

	// The inode of the child file or directory, and its name within the parent.
	Inode fuseops.InodeID
	Name  string

	// The type of the child. The zero value (DT_Unknown) is legal, but means
	// that the kernel will need to call GetAttr when the type is needed.
	Type DirentType
}

// Write the supplied directory entry into the given buffer in the format
// expected in fuseops.ReadDirOp.Data, returning the number of bytes written.
// Return zero if the entry would not fit.
func WriteDirent(buf []byte, d Dirent) (n int) {
	return WriteDirentPlus(buf, nil, d)
}

// Write the supplied directory entry and, optionally, inode entry into the
// given buffer in the format expected in fuseops.ReadDirOp.Data with enabled
// READDIRPLUS capability, returning the number of bytes written.
// Returns zero if the entry would not fit.
func WriteDirentPlus(buf []byte, e *fuseops.ChildInodeEntry, d Dirent) (n int) {
	// We want to write bytes with the layout of fuse_dirent
	// (https://tinyurl.com/4k7y2h9r) in host order. The struct must be aligned
	// according to FUSE_DIRENT_ALIGN (https://tinyurl.com/3m3ewu7h), which
	// dictates 8-byte alignment.
	type fuse_dirent struct {
		ino     uint64
		off     uint64
		namelen uint32
		type_   uint32
		name    [0]byte
	}

	const direntAlignment = 8
	const direntSize = 8 + 8 + 4 + 4

	// Compute the number of bytes of padding we'll need to maintain alignment
	// for the next entry.
	var padLen int
	if len(d.Name)%direntAlignment != 0 {
		padLen = direntAlignment - (len(d.Name) % direntAlignment)
	}

	// Do we have enough room?
	totalLen := direntSize + len(d.Name) + padLen
	if e != nil {
		// READDIRPLUS was added in protocol 7.21, entry attributes were added in 7.9
		// So here EntryOut is always full-length
		totalLen += int(unsafe.Sizeof(fusekernel.EntryOut{}))
	}
	if totalLen > len(buf) {
		return n
	}

	if e != nil {
		out := (*fusekernel.EntryOut)(unsafe.Pointer(&buf[n]))
		fuseops.ConvertChildInodeEntry(e, out)
		n += int(unsafe.Sizeof(fusekernel.EntryOut{}))
	}

	// Write the header.
	de := fuse_dirent{
		ino:     uint64(d.Inode),
		off:     uint64(d.Offset),
		namelen: uint32(len(d.Name)),
		type_:   uint32(d.Type),
	}

	n += copy(buf[n:], (*[direntSize]byte)(unsafe.Pointer(&de))[:])

	// Write the name afterward.
	n += copy(buf[n:], d.Name)

	// Add any necessary padding.
	if padLen != 0 {
		var padding [direntAlignment]byte
		n += copy(buf[n:], padding[:padLen])
	}

	return n
}
