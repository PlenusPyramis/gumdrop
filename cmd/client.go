package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"log"
	"sort"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func GetClient(token string) *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: token,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)
	return client
}

func GetAccount(client *godo.Client) (*godo.Account, error) {
	ctx := context.TODO()
	account, _, err := client.Account.Get(ctx)
	if err != nil {
		return account, err
	}
	if account.Status != "active" {
		return account, errors.New("Account is inactive")
	}
	return account, nil
}

func GetRegions(client *godo.Client) ([]string, map[string]godo.Region, error) {
	ctx := context.TODO()
	opt := &godo.ListOptions{PerPage: 100}
	regions, _, err := client.Regions.List(ctx, opt)
	if err != nil {
		return nil, nil, err
	}
	regionMap := make(map[string]godo.Region)
	for _, r := range regions {
		regionMap[r.Name+" - "+r.Slug] = r
	}
	keys := make([]string, 0, len(regionMap))
	for k := range regionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, regionMap, nil
}

func GetImages(client *godo.Client) ([]string, map[string]godo.Image, error) {
	ctx := context.TODO()
	opt := &godo.ListOptions{}
	imageMap := make(map[string]godo.Image)

	for {
		images, resp, err := client.Images.List(ctx, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, i := range images {
			imageMap[i.Distribution+" - "+i.Name] = i
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, nil, err
		}
		opt.Page = page + 1
	}

	keys := make([]string, 0, len(imageMap))
	for k := range imageMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, imageMap, nil
}

func GetSizes(client *godo.Client) ([]string, map[string]godo.Size, error) {
	ctx := context.TODO()
	opt := &godo.ListOptions{}
	sizeMap := make(map[string]godo.Size)

	for {
		sizes, resp, err := client.Sizes.List(ctx, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, s := range sizes {
			sizeMap[fmt.Sprintf("%s - $%0.2f/month", s.Slug, s.PriceMonthly)] = s
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, nil, err
		}
		opt.Page = page + 1
	}

	keys := make([]string, 0, len(sizeMap))
	for k := range sizeMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, sizeMap, nil
}

func GetUnassignedFloatingIPs(client *godo.Client, region string) ([]string, map[string]godo.FloatingIP, error) {
	ctx := context.TODO()
	opt := &godo.ListOptions{}
	floatingIPMap := make(map[string]godo.FloatingIP)
	fmt.Println("Getting existing Floating IP addresses ... ")
	for {
		floatingIPs, resp, err := client.FloatingIPs.List(ctx, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, i := range floatingIPs {
			if i.Region.Slug == region && i.Droplet == nil {
				floatingIPMap[fmt.Sprintf("%s - %s", i.IP, i.Region.Slug)] = i
			}
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, nil, err
		}
		opt.Page = page + 1
	}

	keys := make([]string, 0, len(floatingIPMap))
	for k := range floatingIPMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, floatingIPMap, nil
}

func GetUnattachedVolumes(client *godo.Client, region string) ([]string, map[string]godo.Volume, error) {
	ctx := context.TODO()
	opt := &godo.ListVolumeParams{ListOptions: &godo.ListOptions{}}
	volumeMap := make(map[string]godo.Volume)
	fmt.Println("Getting existing volumes ... ")
	for {
		volumes, resp, err := client.Storage.ListVolumes(ctx, opt)
		if err != nil {
			return nil, nil, err
		}
		for _, v := range volumes {
			if v.Region.Slug == region {
				volumeMap[fmt.Sprintf("%s - %s", v.Name, v.Region.Slug)] = v
			}
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, nil, err
		}
		opt.ListOptions.Page = page + 1
	}

	keys := make([]string, 0, len(volumeMap))
	for k := range volumeMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, volumeMap, nil
}

func CreateFloatingIP(client *godo.Client, region string) string {
	ctx := context.TODO()
	fmt.Println("Requesting Floating IP address ...")
	floatingIP, _, err := client.FloatingIPs.Create(ctx, &godo.FloatingIPCreateRequest{Region: region})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created Floating IP address (", region, ") : ", floatingIP.IP)
	return floatingIP.IP
}

func CreateVolume(client *godo.Client, region string, name string, size int64) string {
	ctx := context.TODO()
	fmt.Println("Creating volume :", name, "...")
	volume, _, err := client.Storage.CreateVolume(ctx, &godo.VolumeCreateRequest{
		Region: region, Name: name, SizeGigaBytes: size})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created Volume (", region, ") : ", volume.Name, " id=", volume.ID)
	return volume.ID
}
