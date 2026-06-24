// Copyright 2020 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

package sila

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/sila-org/sila/sila/protocols/sila"
	"github.com/sila-org/sila/sila/protocols/snap"
	"github.com/sila-org/sila/p2p"
)

var (
	// errPeerSetClosed is returned if a peer is attempted to be added or removed
	// from the peer set after it has been terminated.
	errPeerSetClosed = errors.New("peerset closed")

	// errPeerAlreadyRegistered is returned if a peer is attempted to be added
	// to the peer set, but one with the same id already exists.
	errPeerAlreadyRegistered = errors.New("peer already registered")

	// errPeerNotRegistered is returned if a peer is attempted to be removed from
	// a peer set, but no peer with the given id exists.
	errPeerNotRegistered = errors.New("peer not registered")

	// errSnapWithoutEth is returned if a peer attempts to connect only on the
	// snap protocol without advertising the sila main protocol.
	errSnapWithoutEth = errors.New("peer connected on snap without compatible sila support")
)

// peerSet represents the collection of active peers currently participating in
// the `sila` protocol, with or without the `snap` extension.
type peerSet struct {
	peers map[string]*silaPeer // Peers connected on the `sila` protocol

	snapWait map[string]chan *snap.Peer // Peers connected on `sila` waiting for their snap extension
	snapPend map[string]*snap.Peer      // Peers connected on the `snap` protocol, but not yet on `sila`

	lock   sync.RWMutex
	closed bool
	quitCh chan struct{} // Quit channel to signal termination
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers:    make(map[string]*silaPeer),
		snapWait: make(map[string]chan *snap.Peer),
		snapPend: make(map[string]*snap.Peer),
		quitCh:   make(chan struct{}),
	}
}

// registerSnapExtension unblocks an already connected `sila` peer waiting for its
// `snap` extension, or if no such peer exists, tracks the extension for the time
// being until the `sila` main protocol starts looking for it.
func (ps *peerSet) registerSnapExtension(peer *snap.Peer) error {
	// Reject the peer if it advertises `snap` without `sila` as `snap` is only a
	// satellite protocol meaningful with the chain selection of `sila`
	if !peer.RunningCap(sila.ProtocolName, sila.ProtocolVersions) {
		return fmt.Errorf("%w: have %v", errSnapWithoutEth, peer.Caps())
	}
	// Ensure nobody can double connect
	ps.lock.Lock()
	defer ps.lock.Unlock()

	id := peer.ID()
	if _, ok := ps.peers[id]; ok {
		return errPeerAlreadyRegistered // avoid connections with the same id as existing ones
	}
	if _, ok := ps.snapPend[id]; ok {
		return errPeerAlreadyRegistered // avoid connections with the same id as pending ones
	}
	// Inject the peer into an `sila` counterpart is available, otherwise save for later
	if wait, ok := ps.snapWait[id]; ok {
		delete(ps.snapWait, id)
		wait <- peer
		return nil
	}
	ps.snapPend[id] = peer
	return nil
}

// waitSnapExtension blocks until all satellite protocols are connected and tracked
// by the peerset.
func (ps *peerSet) waitSnapExtension(peer *sila.Peer) (*snap.Peer, error) {
	// If the peer does not support a compatible `snap`, don't wait
	if !peer.RunningCap(snap.ProtocolName, snap.ProtocolVersions) {
		return nil, nil
	}
	// Ensure nobody can double connect
	ps.lock.Lock()

	id := peer.ID()
	if _, ok := ps.peers[id]; ok {
		ps.lock.Unlock()
		return nil, errPeerAlreadyRegistered // avoid connections with the same id as existing ones
	}
	if _, ok := ps.snapWait[id]; ok {
		ps.lock.Unlock()
		return nil, errPeerAlreadyRegistered // avoid connections with the same id as pending ones
	}
	// If `snap` already connected, retrieve the peer from the pending set
	if snap, ok := ps.snapPend[id]; ok {
		delete(ps.snapPend, id)

		ps.lock.Unlock()
		return snap, nil
	}
	// Otherwise wait for `snap` to connect concurrently
	wait := make(chan *snap.Peer)
	ps.snapWait[id] = wait
	ps.lock.Unlock()

	select {
	case p := <-wait:
		return p, nil
	case <-ps.quitCh:
		ps.lock.Lock()
		delete(ps.snapWait, id)
		ps.lock.Unlock()
		return nil, errPeerSetClosed
	}
}

// registerPeer injects a new `sila` peer into the working set, or returns an error
// if the peer is already known.
func (ps *peerSet) registerPeer(peer *sila.Peer, ext *snap.Peer) error {
	// Start tracking the new peer
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errPeerSetClosed
	}
	id := peer.ID()
	if _, ok := ps.peers[id]; ok {
		return errPeerAlreadyRegistered
	}
	sila := &silaPeer{
		Peer: peer,
	}
	if ext != nil {
		sila.snapExt = &snapPeer{ext}
	}
	ps.peers[id] = sila
	return nil
}

// unregisterPeer removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) unregisterPeer(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errPeerNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// peer retrieves the registered peer with the given id.
func (ps *peerSet) peer(id string) *silaPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// all returns all current peers.
func (ps *peerSet) all() []*silaPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return slices.Collect(maps.Values(ps.peers))
}

// len returns if the current number of `sila` peers in the set. Since the `snap`
// peers are tied to the existence of an `sila` connection, that will always be a
// subset of `sila`.
func (ps *peerSet) len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// snapLen returns the number of `snap` peers whose negotiated version is at
// least minVersion — i.e. peers usable by a state syncer running that version.
// Lower-version snap peers (which the node still serves but cannot sync from)
// are excluded, so they don't get counted toward the reserved snap-peer slots.
func (ps *peerSet) snapLen(minVersion uint) int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var n int
	for _, p := range ps.peers {
		if p.snapExt != nil && p.snapExt.Version() >= minVersion {
			n++
		}
	}
	return n
}

// close disconnects all peers.
func (ps *peerSet) close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	if !ps.closed {
		close(ps.quitCh)
	}
	ps.closed = true
}
