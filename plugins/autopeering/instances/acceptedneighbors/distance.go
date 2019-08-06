package acceptedneighbors

import (
	"hash/fnv"

	"github.com/iotaledger/goshimmer/plugins/autopeering/instances/ownpeer"
	"github.com/iotaledger/goshimmer/plugins/autopeering/saltmanager"
	"github.com/iotaledger/goshimmer/plugins/autopeering/types/peer"
)

var DISTANCE = func(anchor *peer.Peer) func(p *peer.Peer) uint64 {
	return func(p *peer.Peer) uint64 {
		saltedIdentifier := make([]byte, len(anchor.GetIdentity().Identifier)+len(saltmanager.PRIVATE_SALT.GetBytes()))
		copy(saltedIdentifier[0:], anchor.GetIdentity().Identifier)
		copy(saltedIdentifier[len(anchor.GetIdentity().Identifier):], saltmanager.PRIVATE_SALT.GetBytes())

		return hash(saltedIdentifier) ^ hash(p.GetIdentity().Identifier)
	}
}

var OWN_DISTANCE func(p *peer.Peer) uint64

func configureOwnDistance() {
	OWN_DISTANCE = DISTANCE(ownpeer.INSTANCE)
}

func hash(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)

	return h.Sum64()
}
