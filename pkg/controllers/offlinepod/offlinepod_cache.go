package offlinepod

import (
	"container/list"

	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/klog"
)

type Cache struct {
	Name       string
	MaxEntries int
	ll         *list.List
}

func New(maxEntries int, name string) *Cache {
	return &Cache{
		Name:       name,
		MaxEntries: maxEntries,
		ll:         list.New(),
	}
}

func (c *Cache) Add(value *model.OfflinePod) {
	c.ll.PushFront(value)
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
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
	}
}

func (c *Cache) List() []*model.OfflinePod {
	ls := make([]*model.OfflinePod, 0, c.MaxEntries)

	klog.V(4).Infof("name: %s list len: %d", c.Name, c.ll.Len())
	for e := c.ll.Front(); e != nil; e = e.Next() {
		ls = append(ls, e.Value.(*model.OfflinePod))
	}

	return ls
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
