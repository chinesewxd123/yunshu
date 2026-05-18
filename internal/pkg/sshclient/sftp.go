package sshclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
)

// DownloadFile 通过 SFTP 下载远端文件到本地路径。
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("ssh client not connected")
	}
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}

	src, err := sftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	done := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(dst, src)
		done <- copyErr
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// RemoteFileSize 返回远端文件大小。
func (c *Client) RemoteFileSize(remotePath string) (int64, error) {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()
	st, err := sftpClient.Stat(remotePath)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

// RemoveRemoteFile 删除远端文件。
func (c *Client) RemoveRemoteFile(remotePath string) error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	return sftpClient.Remove(remotePath)
}

// WaitRemoteFile 等待远端文件出现且大小稳定（用于 mysqldump 落盘）。
func (c *Client) WaitRemoteFile(ctx context.Context, remotePath string, minSize int64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastSize int64 = -1
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		size, err := c.RemoteFileSize(remotePath)
		if err == nil && size >= minSize {
			if size == lastSize {
				return nil
			}
			lastSize = size
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("remote file not ready: %s", remotePath)
}
