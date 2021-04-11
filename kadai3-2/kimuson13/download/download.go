package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
)

type Range struct {
	low    int
	high   int
	number int
}

func Run(url string, ctx context.Context) error {
	parallel := runtime.NumCPU()
	contentLength, err := checkContentLength(url, ctx)
	if err != nil {
		return err
	}

	if err := Download(url, contentLength, parallel, ctx); err != nil {
		return err
	}

	if err := MergeFile(parallel, contentLength); err != nil {
		return err
	}

	return nil
}

func Download(url string, contentLength, parallel int, ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := os.Mkdir("tempdir", 0755); err != nil {
		return err
	}

	split := contentLength / parallel
	grp, ctx := errgroup.WithContext(ctx)
	for i := 0; i < parallel; i++ {
		r := MakeRange(i, split, parallel, contentLength)
		tempfile := fmt.Sprintf("tempdir/tempfile.%d.%d", parallel, r.number)
		file, err := os.Create(tempfile)
		if err != nil {
			return err
		}
		defer file.Close()
		filename := file.Name()
		grp.Go(func() error {
			err := Requests(r, url, filename)
			return err
		})
	}

	if err := grp.Wait(); err != nil {
		return err
	}

	return nil
}

func checkContentLength(url string, ctx context.Context) (int, error) {
	fmt.Fprintf(os.Stdout, "Start HEAD request to check Content-Length\n")

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	acceptRange := res.Header.Get("Accept-Ranges")
	fmt.Fprintf(os.Stdout, "got: Accept-Ranges: %s\n", acceptRange)
	if acceptRange == "" {
		return 0, errors.New("Accept-Range is not bytes")
	}
	if acceptRange != "bytes" {
		return 0, errors.New("it is not content")
	}

	contentLength := int(res.ContentLength)
	fmt.Fprintf(os.Stdout, "got: Content-Length: %v\n", contentLength)
	if contentLength < 1 {
		return 0, errors.New("it does not have Content-Length")
	}

	return contentLength, nil
}

func MakeRange(i, split, parallel, contentLength int) Range {
	low := split * i
	high := low + split - 1
	if i == parallel-1 {
		high = contentLength
	}

	return Range{
		low:    low,
		high:   high,
		number: i,
	}
}

func Requests(r Range, url, filename string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", r.low, r.high))
	fmt.Fprintf(os.Stdout, "start GET request: bytes=%d-%d\n", r.low, r.high)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	output, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, res.Body)
	if err != nil {
		return err
	}
	return nil
}

func MergeFile(parallel, contentLength int) error {
	fmt.Println("\nmerging files...")
	filename := "paralleldownload"
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fh.Close()

	var n string
	for i := 0; i < parallel; i++ {
		n = fmt.Sprintf("tempdir/tempfile.%d.%d", parallel, i)
		sub, err := os.Open(n)
		if err != nil {
			return err
		}
		_, err = io.Copy(fh, sub)
		if err != nil {
			return err
		}
		sub.Close()
	}
	if err := os.RemoveAll("tempdir"); err != nil {
		return err
	}
	fmt.Println("complete parallel donwload!")
	return nil
}
