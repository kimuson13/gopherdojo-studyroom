package options

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Help     bool   `short:"h" long:"help"`
	Parallel int    `short:"p" long:"parallel"`
	Output   string `short:"o" long:"output"`
	Timeout  int    `short:"t" long:"timuout"`
}

func (opt *Options) parse(argv []string) ([]string, error) {
	p := flags.NewParser(opt, flags.PrintErrors)
	args, err := p.ParseArgs(argv)
	if err != nil {
		os.Stderr.Write(opt.usage())
		return nil, errors.New("it is not comand")
	}
	return args, nil
}

func (opt *Options) usage() []byte {
	buf := bytes.Buffer{}
	msg := "Pdownload 1.0.0, parallel file download client\n"
	fmt.Fprintf(&buf, msg+
		`Usage: Pdownload [options] URL
	Options:
	-h, --help              print usage and exit
	-p, --parallel <num>    split ratio to download file
	-o, --output <filename>	output file to <filename>
	-t, --timeout <seconds> timeout of request for seconds`)
	return buf.Bytes()
}
