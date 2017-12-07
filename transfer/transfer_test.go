package transfer

import (
	"testing"

	"os"

	"fmt"
	"time"

	"log"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/blobporter/pipeline"
	"github.com/Azure/blobporter/sources"
	"github.com/Azure/blobporter/targets"
	"github.com/Azure/blobporter/util"
	"github.com/stretchr/testify/assert"
)

var sourceFiles = make([]string, 1)
var accountName = os.Getenv("ACCOUNT_NAME")
var accountKey = os.Getenv("ACCOUNT_KEY")
var blockSize = uint64(4 * util.MiByte)
var delegate = func(r pipeline.WorkerResult, committedCount int, bufferLevel int) {}
var numOfReaders = 10
var numOfWorkers = 10
var filesPerPipeline = 10

const (
	containerName1 = "bptest"
	containerName2 = "bphttptest"
)

func TestFileToPageHTTPToPage(t *testing.T) {
	container, containerHTTP := getContainers()

	var sourceFile = createPageFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzurePage(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()
	sourceURI, err := createSasTokenURL(sourceFile, container)

	assert.NoError(t, err)

	ap = targets.NewAzurePage(accountName, accountKey, containerHTTP)
	fp = sources.NewHTTP([]string{sourceURI}, []string{sourceFile}, true)
	tfer = NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)
}

func TestFileToPage(t *testing.T) {
	var sourceFile = createPageFile("tb", 1)
	container, _ := getContainers()

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	pt := targets.NewAzurePage(accountName, accountKey, container)

	tfer := NewTransfer(&fp, &pt, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)
}

func TestFileToFile(t *testing.T) {
	var sourceFile = createFile("tb", 1)
	var destFile = "d" + sourceFile

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		TargetAliases:    []string{destFile},
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ft := targets.NewMultiFile(true, numOfWorkers)

	tfer := NewTransfer(&fp, &ft, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)
	os.Remove(destFile)

}

func TestFileToBlob(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)

}
func TestFileToBlobToBlock(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	sourceParams = &sources.AzureBlobParams{
		Container:         container,
		BlobNames:         []string{sourceFile},
		AccountName:       accountName,
		AccountKey:        accountKey,
		CalculateMD5:      true,
		UseExactNameMatch: true,
		FilesPerPipeline:  10,
		//default to always true so blob names are kept
		KeepDirStructure: true}

	source := sources.NewAzureBlob(sourceParams)
	target := targets.NewAzurePage(storageAccountName, storageAccountKey, containerName)

	os.Remove(sourceFile)

}
func TestFileToBlobWithLargeBlocks(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)
	bsize := uint64(16 * util.MiByte)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, bsize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)
}
func TestFilesToBlob(t *testing.T) {
	container, _ := getContainers()

	var sf1 = createFile("tbm", 1)
	var sf2 = createFile("tbm", 1)
	var sf3 = createFile("tbm", 1)
	var sf4 = createFile("tbm", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{"tbm*"},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sf1)
	os.Remove(sf2)
	os.Remove(sf3)
	os.Remove(sf4)
}

func TestFileToBlobHTTPToBlob(t *testing.T) {
	container, containerHTTP := getContainers()

	var sourceFile = createFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()
	sourceURI, err := createSasTokenURL(sourceFile, container)

	assert.NoError(t, err)

	ap = targets.NewAzureBlock(accountName, accountKey, containerHTTP)
	fp = sources.NewHTTP([]string{sourceURI}, []string{sourceFile}, true)
	tfer = NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)
}
func TestFileToBlobHTTPToFile(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()
	sourceURI, err := createSasTokenURL(sourceFile, container)

	assert.NoError(t, err)

	sourceFiles[0] = "d" + sourceFile
	ap = targets.NewMultiFile(true, numOfWorkers)
	fp = sources.NewHTTP([]string{sourceURI}, sourceFiles, true)
	tfer = NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove("d" + sourceFile)
	os.Remove(sourceFile)

}
func TestFileToBlobToFile(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	ap = targets.NewMultiFile(true, numOfWorkers)
	azureBlobParams := &sources.AzureBlobParams{
		Container:         container,
		BlobNames:         []string{sourceFile},
		AccountName:       accountName,
		AccountKey:        accountKey,
		FilesPerPipeline:  filesPerPipeline,
		CalculateMD5:      true,
		UseExactNameMatch: false}
	fp = sources.NewAzureBlob(azureBlobParams)[0]
	tfer = NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(sourceFile)

}
func TestFileToBlobToFileWithAlias(t *testing.T) {
	container, _ := getContainers()
	var sourceFile = createFile("tb", 1)
	var alias = "x" + sourceFile

	sourceParams := &sources.MultiFileParams{
		SourcePatterns:   []string{sourceFile},
		BlockSize:        blockSize,
		TargetAliases:    []string{alias},
		FilesPerPipeline: filesPerPipeline,
		NumOfPartitions:  numOfReaders,
		MD5:              true}

	fp := sources.NewMultiFile(sourceParams)[0]
	ap := targets.NewAzureBlock(accountName, accountKey, container)
	tfer := NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	ap = targets.NewMultiFile(true, numOfWorkers)
	azureBlobParams := &sources.AzureBlobParams{
		Container:         container,
		BlobNames:         []string{alias},
		AccountName:       accountName,
		AccountKey:        accountKey,
		CalculateMD5:      true,
		FilesPerPipeline:  filesPerPipeline,
		UseExactNameMatch: false}
	fp = sources.NewAzureBlob(azureBlobParams)[0]
	tfer = NewTransfer(&fp, &ap, numOfReaders, numOfWorkers, blockSize)
	tfer.StartTransfer(None, delegate)
	tfer.WaitForCompletion()

	os.Remove(alias)
	os.Remove(sourceFile)

}
func deleteContainer(containerName string) {
	_, err := getClient().DeleteContainerIfExists(containerName)
	if err != nil {
		panic(err)
	}
}
func getContainers() (string, string) {
	return createContainer(containerName1), createContainer(containerName2)
}

func createContainer(containerName string) string {
	_, err := getClient().CreateContainerIfNotExists(containerName, storage.ContainerAccessTypePrivate)

	if err != nil {
		panic(err)
	}
	return containerName
}
func getClient() storage.BlobStorageClient {
	client, err := storage.NewBasicClient(accountName, accountKey)

	if err != nil {
		panic(err)
	}

	return client.GetBlobService()

}
func createSasTokenURL(blobName string, container string) (string, error) {
	date := time.Now().UTC().AddDate(0, 0, 3)
	return getClient().GetBlobSASURI(container, blobName, date, "r")
}
func getStorageAccountCreds() *pipeline.StorageAccountCredentials {
	return &pipeline.StorageAccountCredentials{AccountName: accountName, AccountKey: accountKey}
}
func createPageFile(prefix string, sizeInMB int) string {
	fileName := fmt.Sprintf("%v%v", prefix, time.Now().Nanosecond())
	var file *os.File
	var err error

	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	if file, err = os.Create(fileName); os.IsExist(err) {
		if err = os.Remove(fileName); err != nil {
			log.Fatal(err)
		}
		if file, err = os.Create(fileName); err != nil {
			log.Fatal(err)
		}
	}

	b := make([]byte, util.MiByte)

	for i := 0; i < util.MiByte; i++ {
		b[i] = 1
	}

	for i := 0; i < sizeInMB; i++ {
		file.Write(b)
	}

	return fileName
}
func createFile(prefix string, approxSizeInMB int) string {
	fileName := fmt.Sprintf("%v%v", prefix, time.Now().Nanosecond())
	var file *os.File
	var err error

	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	if file, err = os.Create(fileName); os.IsExist(err) {
		if err = os.Remove(fileName); err != nil {
			log.Fatal(err)
		}
		if file, err = os.Create(fileName); err != nil {
			log.Fatal(err)
		}
	}

	b := make([]byte, util.MiByte)

	for i := 0; i < util.MiByte; i++ {
		b[i] = 1
	}

	for i := 0; i < approxSizeInMB; i++ {
		file.Write(b)
	}

	//Add a more data to make the size not a multiple of 1 MB
	file.Write(b[:123])

	return fileName
}
