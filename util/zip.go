package util

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Zip struct {
	src      []string
	dest     string
	progress func(p int)
}

func New(src []string, dest string) *Zip {
	return &Zip{
		src:  src,
		dest: dest,
	}
}

func (z *Zip) Compress(pF func(p int)) error {
	z.progress = pF
	files := []*os.File{}
	for _, p := range z.src {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		files = append(files, f)
	}
	return z.compressFile(files, z.dest)
}

func (z *Zip) compressFile(files []*os.File, dest string) error {
	d, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer d.Close()
	w := zip.NewWriter(d)
	defer w.Close()
	for _, file := range files {
		defer file.Close()
		err := z.compress(file, "", w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (z *Zip) compress(file *os.File, prefix string, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			defer f.Close()
			err = z.compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func Unzip(zipFile string, destDir string, infoFunc func(fileNumber int, progress int)) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()
	fileNumber := len(zipReader.File)
	if infoFunc != nil && fileNumber == 0 {
		infoFunc(fileNumber, 0)
		return nil
	}
	for i, f := range zipReader.File {
		if f.Flags == 0 {
			i := bytes.NewReader([]byte(f.Name))
			decoder := transform.NewReader(i, simplifiedchinese.GB18030.NewDecoder())
			content, _ := ioutil.ReadAll(decoder)
			f.Name = string(content)
		}
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				inFile.Close()
				return err
			}
			buf := make([]byte, 1024*100)
			for {
				n, err := inFile.Read(buf)
				if err != nil && err != io.EOF {
					return err
				}
				if n == 0 {
					break
				}
				_, err2 := outFile.Write(buf[:n])
				if err2 != nil {
					return err2
				}
				if err == io.EOF {
					break
				}
			}
			outFile.Close()
			inFile.Close()
		}
		if infoFunc != nil {
			infoFunc(len(zipReader.File), i)
		}
	}
	return nil
}
