package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type ResourceSet struct {
	Id       string            `json:"id"`
	Limits   ResourceSetLimits `json:"limits"`
	Name     string            `json:"name"`
	Subjects []string          `json:"subjects"`
	Objects  []string          `json:"objects"`
}

type ResourceSetLimits struct {
	Cpus   ResourceSetLimit
	Memory ResourceSetLimit
	Disk   ResourceSetLimit
}

type ResourceSetLimit struct {
	Available int
	Total     int
}

func (c Client) GetResourceSets() ([]ResourceSet, error) {
	return c.makeResourceSetGetAllCall()
}

func (c Client) GetResourceSetById(id string) (*ResourceSet, error) {
	resourceSets, err := c.GetResourceSet(ResourceSet{
		Id: id,
	})

	if err != nil {
		return nil, err
	}

	l := len(resourceSets)
	if l != 1 {
		return nil, errors.New(fmt.Sprintf("found `%d` resource set(s) with id `%s`: %v", l, id, resourceSets))
	}

	return &resourceSets[0], nil
}

func (c Client) GetResourceSet(rsReq ResourceSet) ([]ResourceSet, error) {
	resourceSets, err := c.makeResourceSetGetAllCall()

	if err != nil {
		return nil, err
	}
	rsRv := []ResourceSet{}
	found := false
	for _, rs := range resourceSets {
		if rsReq.Id == rs.Id {
			found = true
			rsRv = append(rsRv, rs)
		}

		if rsReq.Name == rs.Name {
			found = true
			rsRv = append(rsRv, rs)
		}
	}

	if !found {
		return rsRv, NotFound{}
	}

	return rsRv, nil
}

func (c Client) makeResourceSetGetAllCall() ([]ResourceSet, error) {

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	var res struct {
		ResourceSets []ResourceSet `json:"-"`
	}
	params := map[string]interface{}{
		"id": "dummy",
	}
	err := c.Call(ctx, "resourceSet.getAll", params, &res.ResourceSets)
	fmt.Printf("[DEBUG] Calling resourceSet.getAll received response: %+v with error: %v\n", res, err)

	if err != nil {
		return nil, err
	}

	return res.ResourceSets, nil
}

func (c Client) CreateResourceSet(rsReq ResourceSet) (*ResourceSet, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	rs := ResourceSet{}
	params := map[string]interface{}{
		"name":     rsReq.Name,
		"subjects": rsReq.Subjects,
		"objects":  rsReq.Objects,
		"limits":   rsReq.Limits,
	}
	err := c.Call(ctx, "resourceSet.create", params, &rs)
	fmt.Printf("[DEBUG] Calling resourceSet.create with params: %v returned: %+v with error: %v\n", params, rs, err)

	if err != nil {
		return nil, err
	}

	return &rs, err
}

func (c Client) DeleteResourceSet(rsReq ResourceSet) error {

	id := rsReq.Id
	if id == "" {
		rs, err := c.GetResourceSet(rsReq)

		if err != nil {
			return err
		}

		if len(rs) > 1 {
			return errors.New(fmt.Sprintf("refusing to delete resource set since `%d` resource sets were returned: %v", len(rs), rs))
		}

		id = rs[0].Id
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	var success bool
	params := map[string]interface{}{
		"id": id,
	}
	err := c.Call(ctx, "resourceSet.delete", params, &success)
	fmt.Printf("[DEBUG] Calling resourceSet.delete call successful: %t with error: %v\n", success, err)

	return err
}

func RemoveResourceSetsWithNamePrefix(rsNamePrefix string) func(string) error {
	return func(_ string) error {
		fmt.Println("[DEBUG] Running sweeper")
		c, err := NewClient(GetConfigFromEnv())
		if err != nil {
			return fmt.Errorf("error getting client: %s", err)
		}

		rss, err := c.GetResourceSets()
		if err != nil {
			return fmt.Errorf("error getting resource sets: %s", err)
		}
		for _, rs := range rss {
			if strings.HasPrefix(rs.Name, rsNamePrefix) {
				err := c.DeleteResourceSet(ResourceSet{Id: rs.Id})

				if err != nil {
					log.Printf("error destroying resource set `%s` during sweep: %s", rs.Name, err)
				}
			}
		}
		return nil
	}
}