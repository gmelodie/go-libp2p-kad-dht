package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	u "github.com/ipfs/go-ipfs-util"
	peer "github.com/libp2p/go-libp2p-peer"
)

func TestProviderManager(t *testing.T) {
	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())
	a := cid.NewCidV0(u.Hash([]byte("test")))
	p.AddProvider(ctx, a, peer.ID("testingprovider"))
	resp := p.GetProviders(ctx, a)
	if len(resp) != 1 {
		t.Fatal("Could not retrieve provider.")
	}
	p.proc.Close()
}

func TestProvidersDatastore(t *testing.T) {
	old := lruCacheSize
	lruCacheSize = 10
	defer func() { lruCacheSize = old }()

	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())
	defer p.proc.Close()

	friend := peer.ID("friend")
	var cids []*cid.Cid
	for i := 0; i < 100; i++ {
		c := cid.NewCidV0(u.Hash([]byte(fmt.Sprint(i))))
		cids = append(cids, c)
		p.AddProvider(ctx, c, friend)
	}

	for _, c := range cids {
		resp := p.GetProviders(ctx, c)
		if len(resp) != 1 {
			t.Fatal("Could not retrieve provider.")
		}
		if resp[0] != friend {
			t.Fatal("expected provider to be 'friend'")
		}
	}
}

func TestProvidersSerialization(t *testing.T) {
	dstore := ds.NewMapDatastore()

	k := cid.NewCidV0(u.Hash(([]byte("my key!"))))
	p1 := peer.ID("peer one")
	p2 := peer.ID("peer two")
	pt1 := time.Now()
	pt2 := pt1.Add(time.Hour)

	err := writeProviderEntry(dstore, k, p1, pt1)
	if err != nil {
		t.Fatal(err)
	}

	err = writeProviderEntry(dstore, k, p2, pt2)
	if err != nil {
		t.Fatal(err)
	}

	pset, err := loadProvSet(dstore, k)
	if err != nil {
		t.Fatal(err)
	}

	lt1, ok := pset.set[p1]
	if !ok {
		t.Fatal("failed to load set correctly")
	}

	if pt1 != lt1 {
		t.Fatal("time wasnt serialized correctly")
	}

	lt2, ok := pset.set[p2]
	if !ok {
		t.Fatal("failed to load set correctly")
	}

	if pt2 != lt2 {
		t.Fatal("time wasnt serialized correctly")
	}
}

func TestProvidesExpire(t *testing.T) {
	pval := ProvideValidity
	cleanup := defaultCleanupInterval
	ProvideValidity = time.Second / 2
	defaultCleanupInterval = time.Second / 2
	defer func() {
		ProvideValidity = pval
		defaultCleanupInterval = cleanup
	}()

	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())

	peers := []peer.ID{"a", "b"}
	var cids []*cid.Cid
	for i := 0; i < 10; i++ {
		c := cid.NewCidV0(u.Hash([]byte(fmt.Sprint(i))))
		cids = append(cids, c)
		p.AddProvider(ctx, c, peers[0])
		p.AddProvider(ctx, c, peers[1])
	}

	for i := 0; i < 10; i++ {
		out := p.GetProviders(ctx, cids[i])
		if len(out) != 2 {
			t.Fatal("expected providers to still be there")
		}
	}

	time.Sleep(time.Second)
	for i := 0; i < 10; i++ {
		out := p.GetProviders(ctx, cids[i])
		if len(out) > 0 {
			t.Fatal("expected providers to be cleaned up, got: ", out)
		}
	}

	if p.providers.Len() != 0 {
		t.Fatal("providers map not cleaned up")
	}

	allprovs, err := p.getAllProvKeys()
	if err != nil {
		t.Fatal(err)
	}

	if len(allprovs) != 0 {
		t.Fatal("expected everything to be cleaned out of the datastore")
	}
}

/* This can be used for profiling. Keeping it commented out for now to avoid incurring extra CI time
func TestLargeProvidersSet(t *testing.T) {
	old := lruCacheSize
	lruCacheSize = 10
	defer func() { lruCacheSize = old }()

	dirn, err := ioutil.TempDir("", "provtest")
	if err != nil {
		t.Fatal(err)
	}

	opts := &lds.Options{
		NoSync:      true,
		Compression: 1,
	}
	lds, err := lds.NewDatastore(dirn, opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = lds

	defer func() {
		os.RemoveAll(dirn)
	}()

	ctx := context.Background()
	var peers []peer.ID
	for i := 0; i < 3000; i++ {
		peers = append(peers, peer.ID(fmt.Sprint(i)))
	}

	mid := peer.ID("myself")
	p := NewProviderManager(ctx, mid, lds)
	defer p.proc.Close()

	var cids []*cid.Cid
	for i := 0; i < 1000; i++ {
		c := cid.NewCidV0(u.Hash([]byte(fmt.Sprint(i))))
		cids = append(cids, c)
		for _, pid := range peers {
			p.AddProvider(ctx, c, pid)
		}
	}

	for _, c := range cids {
		_ = p.GetProviders(ctx, c)
	}

}
*/
