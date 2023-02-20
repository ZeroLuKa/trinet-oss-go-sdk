package ossClient

import (
	"context"
	"fmt"
	"github.com/ZeroLuKa/trinet-oss-go-sdk/pkg/credentials"
	"github.com/ZeroLuKa/trinet-oss-go-sdk/pkg/tags"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

//RandomStr 随机生成字符串
func RandomStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()_+{}|:<>;."
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// 测试基础流程
func TestClient_BaseMergePartUpload(t *testing.T) {
	// 测试数据生成
	fileNum := 100
	testData := make(map[string]string)
	totalSize := 0
	for i := 1; i < fileNum; i++ {
		testData[strconv.Itoa(i)] = RandomStr(i)
		totalSize += len(testData[strconv.Itoa(i)])
	}

	// 创建测试的bucket
	id := ""
	bucket := "test-merge"
	opts := &Options{
		Creds: credentials.NewStaticV4("minioadmin", "minioadmin", ""),
	}
	client, err := New("127.0.0.1:19000", opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = client.MakeBucket(context.Background(), bucket, MakeBucketOptions{})
	defer client.RemoveBucket(context.Background(), bucket)

	// 合并上传
	p, err := client.InitMergePartUpload(id, bucket)
	if err != nil {
		t.Fatal(err)
	}
	id = p.ID

	for i := 1; i < fileNum; i++ {
		_, err := p.UploadMergePart(strconv.Itoa(i), strings.NewReader(testData[strconv.Itoa(i)]))
		if err != nil {
			t.Fatal(err)
		}
		//println(uploadInfo.TotalSize, uploadInfo.ObjectNum)
	}
	err = p.CompleteMergePartUpload(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer client.DeleteMergeID(context.Background(), id, bucket)

	// 获取对象
	data, meta, err := client.GetObjectWithID(context.Background(), id, bucket, "1")
	if err != nil {
		t.Fatal(err)
	}
	if meta.TotalSize != int64(totalSize) {
		t.Fatal("total size error")
	}
	s, err := io.ReadAll(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(s) != testData["1"] {
		t.Fatal("get data error")
	}
	//t.Log(1, "bingo")
	data.Close()

	// 使用元数据缓存获取对象
	for i := 2; i < fileNum; i += 1 {
		key := strconv.Itoa(i)
		data, err = client.GetObjectWithIndex(context.Background(), id, bucket, key, meta)
		if err != nil {
			t.Fatal(err)
		}
		s, err = io.ReadAll(data)
		if err != nil {
			t.Fatal(err)
		}
		if string(s) != testData[key] {
			t.Fatal("get data error")
		}
		//t.Log(key, "bingo")
		data.Close()
	}

	// 测试标签
	tagMap := make(map[string]string)
	tagMap["tag1"] = "v1"
	tagMap["tag2"] = "v2"
	otags, err := tags.NewTags(tagMap, true)
	if err != nil {
		t.Fatal(err)
	}
	err = client.PutMergeObjectTagging(context.Background(), bucket, id, otags, PutObjectTaggingOptions{})
	if err != nil {
		t.Fatal(err)
	}

	ret, err := client.GetMergeObjectTagging(context.Background(), bucket, id, GetObjectTaggingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range tagMap {
		if _, ok := ret.ToMap()[k]; !ok || ret.ToMap()[k] != v {
			t.Fatal("tag check success")
		}
	}

}

func TestClient_Vacancy(t *testing.T) {
	// 测试数据生成
	fileNum := 100
	testData := make(map[string]string)
	totalSize := 0
	for i := 1; i < fileNum; i++ {
		testData[strconv.Itoa(i)] = RandomStr(i)
		totalSize += len(testData[strconv.Itoa(i)])
	}

	// 创建测试的bucket
	id := ""
	bucket := "test-merge"
	opts := &Options{
		Creds: credentials.NewStaticV4("minioadmin", "minioadmin", ""),
	}
	client, err := New("127.0.0.1:19000", opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = client.MakeBucket(context.Background(), bucket, MakeBucketOptions{})
	defer client.RemoveBucket(context.Background(), bucket)

	// 合并上传
	p, err := client.InitMergePartUpload(id, bucket)
	if err != nil {
		t.Fatal(err)
	}
	id = p.ID

	for i := 1; i < fileNum; i++ {
		_, err := p.UploadMergePart(strconv.Itoa(i), strings.NewReader(testData[strconv.Itoa(i)]))
		if err != nil {
			t.Fatal(err)
		}
	}
	err = p.CompleteMergePartUpload(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer client.DeleteMergeID(context.Background(), id, bucket)

	_, meta, err := client.GetObjectWithID(context.Background(), id, bucket, "1")
	if err != nil {
		t.Fatal(err)
	}

	tagMap := make(map[string]string)
	tagMap["tag1"] = "v1"
	tagMap["tag2"] = "v2"
	otags, err := tags.NewTags(tagMap, true)
	if err != nil {
		t.Fatal(err)
	}
	err = client.PutMergeObjectTagging(context.Background(), bucket, id, otags, PutObjectTaggingOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// 制造空洞
	deleteIdx := 1
	for i := 1; i < fileNum; i++ {
		err = client.DeleteObjectWithId(context.Background(), id, bucket, strconv.Itoa(i))
		if err != nil {
			t.Fatal(err)
		}
		meta, err = client.GetObjectIndexInfo(context.Background(), id, bucket)
		if meta.VacancySize*100/meta.TotalSize > 66 || deleteIdx == fileNum-2 {
			deleteIdx = i
			break
		}
	}

	// 手动合并空洞
	targetUrl := fmt.Sprintf("http://127.0.0.1:19000/minio/admin/v3/manual-merge-vacancy")
	req, err := http.NewRequest("PUT", targetUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal(err)
	}

	// 测试合并后数据正确性
	for i := 1; i < fileNum; i += 1 {
		key := strconv.Itoa(i)
		data, _, err := client.GetObjectWithID(context.Background(), id, bucket, key)
		if err != nil {
			if i <= deleteIdx && err.Error() == "object not found" {
				continue
			} else {
				t.Fatal(err)
			}
		}
		s, err := io.ReadAll(data)
		if err != nil {
			t.Fatal(err)
		}
		if string(s) != testData[key] {
			t.Fatal("get data error", i+1)
		}
		//t.Log(key, "test bingo")
		data.Close()
	}

	// 验证合并后标签信息正确性
	ret, err := client.GetMergeObjectTagging(context.Background(), bucket, id, GetObjectTaggingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range tagMap {
		if _, ok := ret.ToMap()[k]; !ok || ret.ToMap()[k] != v {
			t.Fatal("tag check success")
		}
	}
}