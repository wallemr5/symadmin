package offlinepod

import (
	"container/list"

	"sync"

	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/klog"
)

type Cache struct {
	Name       string
	MaxEntries int32
	ll         *list.List
	impl       *offlinepodImpl
	sync.RWMutex
}

func New(maxEntries int32, name string, impl *offlinepodImpl) *Cache {
	return &Cache{
		Name:       name,
		MaxEntries: maxEntries,
		ll:         list.New(),
		impl:       impl,
	}
}

func (c *Cache) SetMaxEntries(maxEntries int32) {
	c.Lock()
	defer c.Unlock()
	c.MaxEntries = maxEntries
}

func (c *Cache) GetMaxEntries() int32 {
	c.RLock()
	defer c.RUnlock()
	return c.MaxEntries
}

func (c *Cache) Add(value *model.OfflinePod) {
	c.ll.PushFront(value)
	maxEntries := c.GetMaxEntries()
	if maxEntries != 0 && c.ll.Len() > int(maxEntries) {
		c.removeOldest()
	}
}

func (c *Cache) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		obj := ele.Value.(*model.OfflinePod)
		klog.V(4).Infof("removeOldest name: %s, podIP: %s, hostIP: %s, offlinetime: %v",
			obj.Name, obj.PodIP, obj.HostIP, obj.OfflineTime)
		c.ll.Remove(ele)
		c.impl.PutOfflinePod(obj)
	}
}

func (c *Cache) List() []*model.OfflinePod {
	ls := make([]*model.OfflinePod, 0, c.GetMaxEntries())

	klog.V(4).Infof("name: %s list len: %d", c.Name, c.ll.Len())
	for e := c.ll.Front(); e != nil; e = e.Next() {
		ls = append(ls, e.Value.(*model.OfflinePod))
	}

	return ls
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
