// The following environment variables, if set, will be used:
//
//	* GOSWFS_MASTER_URL
//  * GOSWFS_SCHEME
//	* GOSWFS_MEDIUM_FILE
//	* GOSWFS_SMALL_FILE
//  * GOSWFS_FILER_URL
//
package goseaweedfs

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

var sw *Seaweed

var MediumFile, SmallFile string

func init() {
	// check master url
	if masterURL := os.Getenv("GOSWFS_MASTER_URL"); masterURL != "" {
		scheme := os.Getenv("GOSWFS_SCHEME")
		if scheme == "" {
			scheme = "http"
		}

		var filer []string
		if _filer := os.Getenv("GOSWFS_FILER_URL"); _filer != "" {
			filer = []string{_filer}
		}

		sw = NewSeaweed(scheme, masterURL, filer, 2*1024*1024, 5*time.Minute)
	}

	MediumFile = os.Getenv("GOSWFS_MEDIUM_FILE")
	SmallFile = os.Getenv("GOSWFS_SMALL_FILE")

	time.Sleep(10 * time.Second)
}

func TestUploadLookupserverReplaceDeleteFile(t *testing.T) {
	if sw == nil || MediumFile == "" {
		return
	}

	for i := 1; i <= 2; i++ {
		_, _, fID, err := sw.UploadFile(MediumFile, "", "")
		if err != nil {
			t.Fatal(err)
		}

		//
		if _, err := sw.LookupServerByFileID(fID, nil, true); err != nil {
			t.Fatal(err)
		}

		//
		if _, err := sw.LookupFileID(fID, nil, true); err != nil {
			t.Fatal(err)
		}

		//
		if err := sw.ReplaceFile(fID, SmallFile, false); err != nil {
			t.Fatal(err)
			return
		}

		//
		if err := sw.ReplaceFile(fID, SmallFile, true); err != nil {
			t.Fatal(err)
			return
		}

		if err = sw.DeleteFile(fID, nil); err != nil {
			t.Fatal(err)
			return
		}

		// test upload file
		fh, err := os.Open(MediumFile)
		if err != nil {
			t.Fatal(err)
		}
		defer fh.Close()

		var size int64
		if fi, fiErr := fh.Stat(); fiErr != nil {
			t.Fatal(fiErr)
		} else {
			size = fi.Size()
		}

		if _, fID, err = sw.Upload(fh, "test.txt", size, "col", ""); err != nil {
			t.Fatal(err)
		}

		// Replace with small file
		fs, err := os.Open(SmallFile)
		if err != nil {
			t.Fatal(err)
		}
		defer fs.Close()
		if fi, fiErr := fs.Stat(); fiErr != nil {
			t.Fatal(fiErr)
		} else {
			size = fi.Size()
		}

		if err := sw.Replace(fID, fs, "ta.txt", size, "", "", false); err != nil {
			t.Fatal(err)
			return
		}

		// finally delete
		if err = sw.DeleteFile(fID, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBatchUploadFiles(t *testing.T) {
	if sw == nil {
		return
	}

	if MediumFile != "" && SmallFile != "" {
		_, err := sw.BatchUploadFiles([]string{MediumFile, SmallFile}, "", "")
		if err != nil {
			t.Fatal(err)
		}
	} else if MediumFile != "" {
		_, err := sw.BatchUploadFiles([]string{MediumFile, MediumFile}, "", "")
		if err != nil {
			t.Fatal(err)
		}
	} else if SmallFile != "" {
		_, err := sw.BatchUploadFiles([]string{SmallFile, SmallFile}, "", "")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLookup(t *testing.T) {
	if sw == nil {
		return
	}

	_, err := sw.Lookup("1", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = sw.LookupNoCache("1", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGrowAndGC(t *testing.T) {
	if sw == nil {
		return
	}

	fmt.Println(sw.Grow(50+rand.Int()%14, "imgs", "000", "dc1"))

	sw.GC(1024 * 1024)
}

func TestLookupVolumeIDs(t *testing.T) {
	if sw == nil {
		return
	}

	if _, err := sw.LookupVolumeIDs([]string{"50", "51", "1"}); err != nil {
		t.Fatal(err)
	}
}

func TestStatus(t *testing.T) {
	if sw == nil {
		return
	}

	if _, err := sw.Status(); err != nil {
		t.Fatal(err)
	}
}

func TestClusterStatus(t *testing.T) {
	if sw == nil {
		return
	}

	_, err := sw.ClusterStatus()
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestSubmit(t *testing.T) {
	if sw == nil {
		return
	}

	if SmallFile != "" {
		if _, err := sw.Submit(SmallFile, "", ""); err != nil {
			t.Fatal(err)
			return
		}
	}
}

func TestDeleteChunks(t *testing.T) {
	if sw == nil {
		return
	}

	if MediumFile != "" {
		cm, _, _, err := sw.UploadFile(MediumFile, "", "")
		if err != nil {
			t.Fatal(err)
		}

		if err = sw.DeleteChunks(cm, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFiler(t *testing.T) {
	if sw == nil || sw.Filers == nil || len(sw.Filers) == 0 {
		return
	}

	// test with prefix /
	filer := sw.Filers[0]
	if uploadResult, err := filer.UploadFile(SmallFile, "/js/test.txt", "", ""); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(uploadResult)
	}

	if dir, err := filer.Dir("/js/"); err != nil {
		t.Fatal(err)
	} else {
		if dir.Files == nil || len(dir.Files) == 0 {
			t.Fatal(fmt.Errorf("Directory js contains no file"))
		}

		// check directory
		contain := false
		for _, v := range dir.Files {
			if v.Name == "test.txt" {
				contain = true
				break
			}
		}
		if !contain {
			t.Fatal(fmt.Errorf("Directory js does not contain test.txt"))
		}

		// try to delete this file
		if err := filer.Delete("/js/test.txt"); err != nil {
			t.Fatal(err)
		}
	}

	// test with non prefix /
	filer = sw.Filers[0]
	if uploadResult, err := filer.UploadFile(SmallFile, "jsx/test1.jsx", "", ""); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(uploadResult)
	}

	if dir, err := filer.Dir("jsx"); err != nil {
		t.Fatal(err)
	} else {
		if dir.Files == nil || len(dir.Files) == 0 {
			t.Fatal(fmt.Errorf("Directory js contains no file"))
		}

		// check directory
		contain := false
		for _, v := range dir.Files {
			if v.Name == "test1.jsx" {
				contain = true
				break
			}
		}
		if !contain {
			t.Fatal(fmt.Errorf("Directory jsx does not contain test1.jsx"))
		}

		// try to delete this file
		if err := filer.Delete("jsx/test1.jsx"); err != nil {
			t.Fatal(err)
		}
	}
}
