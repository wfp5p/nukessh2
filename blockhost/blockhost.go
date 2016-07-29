package blockhost

import (
	"fmt"
	"os"
	"nukessh/nukedb"
	"strings"
	"time"
	"database/sql" // just to get ErrNoRows

	"github.com/gmccue/go-ipset"

	"log"
)

type BlockHost struct {
	nukeDB *nukedb.NukeDB
	ips *ipset.IPSet
	setname string
	blocktime time.Duration
}

func New(dbname string, setname string, d time.Duration) (*BlockHost, error) {

	if uid := os.Geteuid(); uid != 0 {
		return nil, fmt.Errorf("blockhost: not running with root privs")
	}

	var bh BlockHost
	bh.setname = setname
	bh.blocktime = d

	if s, err := ipset.New(); err != nil {
		return nil, err
	} else {
		bh.ips = s
	}

	if err := bh.ips.Create(setname, "hash:ip"); err != nil {
		if strings.Contains(err.Error(), "set with the same name already exists") {
			bh.ips.Flush(setname)
		} else {
			return nil, fmt.Errorf("blockhost: error %s when trying to create ipset",err)
		}
	}

	if ndb, err := nukedb.New(dbname); err != nil {
		return nil, err
	} else {
		bh.nukeDB = ndb
	}

	return &bh, nil
}

func (bh BlockHost) Close() {
	// destroy the set
	_ = bh.ips.Destroy(bh.setname)

	// close the SQL connection
	_ = bh.nukeDB.Close()
}

func (bh *BlockHost) addtoset(ip string) error {
	return bh.ips.AddUnique(bh.setname, ip)
}

func (bh *BlockHost) ipinset(ip string) error {
	return bh.ips.Test(bh.setname, ip)
}

// add all active blocks to the set
func (bh *BlockHost) BlockActives() error {
	ips, err := bh.nukeDB.GetActive(time.Now())
	if err != nil {
		return err
	}

	for _, ip := range ips {
		log.Printf("BlockActives blocking %v\n", ip)
		if err := bh.addtoset(ip); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func (bh *BlockHost) delfromset(ip string) error {
	if err := bh.ips.Delete(bh.setname, ip); err != nil {
		if strings.Contains(err.Error(), "it's not added") {
			return nil
		} else {
			return err
		}
	}
	return nil
}

// remove expired hosts from set
func (bh *BlockHost) ExpireDB() error {
	ips, err := bh.nukeDB.GetExpires(time.Now())
	if err != nil {
		return err
	}

	for _, ip := range ips {

		log.Printf("ExpireDB unblocking expired ip %v\n", ip)

		if err := bh.delfromset(ip); err != nil {
			log.Fatal(err)
		}

		if err := bh.nukeDB.ClearExpire(ip); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func (bh *BlockHost) BlockHost(ip string) error {

	// test if it's already blocked
	if err := bh.ipinset(ip); err == nil {
		return nil
	}

	var blocks int
	if r, err := bh.nukeDB.GetInfo(ip); err != nil && err != sql.ErrNoRows  {
		log.Fatal(err)
	} else {
		blocks = r.Blocks
	}

	newexpire := time.Now().Add(bh.blocktime * (1 << uint(blocks)))
	log.Printf("blocking %v until %v\n", ip, newexpire)
	blocks++

	if err := bh.nukeDB.Insert(ip, newexpire, blocks); err != nil {
		return err
	}

	if err := bh.addtoset(ip); err != nil {
		log.Fatal(err)
	}

	return nil
}

