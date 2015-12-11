package inmem

import (
	"testing"

	"github.com/squaremo/ambergreen/common/store"
	"github.com/squaremo/ambergreen/common/store/test"
)

// Test the in-memory mock Store

type testableInMemStore struct {
	store.Store
}

func (tims *testableInMemStore) Reset(t *testing.T) {
	tims.Store = NewInMemStore()
}

func TestInMemStore(t *testing.T) {
	test.RunStoreTestSuite(&testableInMemStore{}, t)
}