// Copyright 2021 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memtable

import (
	"errors"
	"fmt"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage"
	fb "github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/db/factories/base"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/layout/table/v1/iface"
	imem "github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/memtable/v1/base"
	"sync"
)

// manager is the collection manager, it's global,
// created when open db
type manager struct {
	sync.RWMutex

	// opts is the options of aoe
	opts *storage.Options

	// collections are containers of managed collection
	collections map[uint64]imem.ICollection

	// TableData is Table's metadata in memory
	//tableData   iface.ITableData

	// factory is the factory that produces
	// different types of the collection
	factory fb.CollectionFactory
}

var (
	_ imem.IManager = (*manager)(nil)
)

func NewManager(opts *storage.Options, factory fb.MutFactory) *manager {
	m := &manager{
		opts:        opts,
		collections: make(map[uint64]imem.ICollection),
	}
	if factory == nil {
		m.factory = m.createCollection
	} else {
		if factory.GetType() == fb.MUTABLE {
			m.factory = m.createMutCollection
		} else {
			m.factory = m.createCollection
		}
	}
	return m
}

func (m *manager) CollectionIDs() map[uint64]uint64 {
	ids := make(map[uint64]uint64)
	m.RLock()
	for k, _ := range m.collections {
		ids[k] = k
	}
	m.RUnlock()
	return ids
}

func (m *manager) WeakRefCollection(id uint64) imem.ICollection {
	m.RLock()
	c, ok := m.collections[id]
	m.RUnlock()
	if !ok {
		return nil
	}
	return c
}

func (m *manager) StrongRefCollection(id uint64) imem.ICollection {
	m.RLock()
	c, ok := m.collections[id]
	if ok {
		c.Ref()
	}
	m.RUnlock()
	if !ok {
		return nil
	}
	return c
}

func (m *manager) String() string {
	m.RLock()
	defer m.RUnlock()
	s := fmt.Sprintf("<MTManager>(TableCnt=%d)", len(m.collections))
	for _, c := range m.collections {
		s = fmt.Sprintf("%s\n\t%s", s, c.String())
	}
	return s
}

func (m *manager) createCollection(td iface.ITableData) imem.ICollection {
	return NewCollection(td, m.opts)
}

func (m *manager) createMutCollection(td iface.ITableData) imem.ICollection {
	return newMutableCollection(m, td)
}

func (m *manager) RegisterCollection(td interface{}) (c imem.ICollection, err error) {
	m.Lock()
	tableData := td.(iface.ITableData)
	_, ok := m.collections[tableData.GetID()]
	if ok {
		m.Unlock()
		return nil, errors.New("logic error")
	}
	c = m.factory(tableData)
	m.collections[tableData.GetID()] = c
	m.Unlock()
	c.Ref()
	return c, err
}

func (m *manager) UnregisterCollection(id uint64) (c imem.ICollection, err error) {
	m.Lock()
	c, ok := m.collections[id]
	if ok {
		delete(m.collections, id)
	} else {
		m.Unlock()
		return nil, errors.New("logic error")
	}
	m.Unlock()
	return c, err
}
