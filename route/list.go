package route

import (
	"container/list"
	"sort"
	"time"
)

type EphemeralItem struct {
	*Item
	Expired time.Time
}

type EphemeralItems struct {
	list *list.List
}

func NewEphemeralItems() *EphemeralItems {
	return &EphemeralItems{
		list: list.New(),
	}
}

func (e *EphemeralItems) Find(cidr string) *list.Element {
	for elem := e.list.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*EphemeralItem).CIDR == cidr {
			return elem
		}
	}
	return nil
}

func (e *EphemeralItems) Len() int {
	return e.list.Len()
}

func (e *EphemeralItems) Remove(cidr string) *EphemeralItem {
	elem := e.Find(cidr)
	if elem != nil {
		e.list.Remove(elem)
		return elem.Value.(*EphemeralItem)
	}
	return nil
}

func (e *EphemeralItems) Add(i *EphemeralItem) {
	for elem := e.list.Front(); elem != nil; elem = elem.Next() {
		if i.Expired.Before(elem.Value.(*EphemeralItem).Expired) {
			e.list.InsertBefore(i, elem)
			return
		}
	}
	e.list.PushBack(i)
}

func (e *EphemeralItems) GetFront() *EphemeralItem {
	elem := e.list.Front()
	if elem == nil {
		return nil
	}
	return elem.Value.(*EphemeralItem)
}

// -----------------------------------------------------------------------------
type Items []Item

func (is *Items) Append(i *Item) error {
	for _, it := range *is {
		if it.CIDR == i.CIDR {
			return ErrRouteItemExists.Format(i.CIDR)
		}
	}
	*is = append(*is, *i)
	return nil
}

func (is *Items) Len() int {
	return len(*is)
}

func (is Items) Less(i, j int) bool {
	return is[i].CIDR > is[j].CIDR
}

func (is Items) Swap(i, j int) {
	is[i], is[j] = is[j], is[i]
}

func (is Items) Find(cidr string) int {
	for idx, i := range is {
		if i.CIDR == cidr {
			return idx
		}
	}
	return -1
}

func (is *Items) Remove(cidr string) *Item {
	idx := is.Find(cidr)
	if idx == 0 {
		ret := &(*is)[idx]
		*is = (*is)[1:]
		return ret
	} else if idx > 0 {
		ret := &(*is)[idx]
		*is = append((*is)[:idx], (*is)[idx+1:]...)
		return ret
	} else {
		return nil
	}
}

func (is *Items) Sort() {
	sort.Sort(is)
}
