package targets

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/blobporter/pipeline"
	"github.com/Azure/blobporter/util"
)

////////////////////////////////////////////////////////////
///// AzurePage Target
////////////////////////////////////////////////////////////

//AzurePage represents an Azure Block target
type AzurePage struct {
	Creds         *pipeline.StorageAccountCredentials
	Container     string
	StorageClient *storage.BlobStorageClient
}

//NewAzurePage creates a new Azure Block target
func NewAzurePage(accountName string, accountKey string, container string) pipeline.TargetPipeline {
	util.CreateContainerIfNotExists(container, accountName, accountKey)
	creds := pipeline.StorageAccountCredentials{AccountName: accountName, AccountKey: accountKey}
	client := util.GetBlobStorageClient(creds.AccountName, creds.AccountKey)
	return &AzurePage{Creds: &creds, Container: container, StorageClient: &client}
}

//Page blobs limits and units

//PageSize page size for page blobs
const PageSize uint64 = 512
const maxPageSize uint64 = 4 * util.MB
const maxPageBlobSize uint64 = 8 * util.TB

//PreProcessSourceInfo implementation of PreProcessSourceInfo from the pipeline.TargetPipeline interface.
//initializes the page blob.
func (t *AzurePage) PreProcessSourceInfo(source *pipeline.SourceInfo, blockSize uint64) (err error) {
	size := int64(source.Size)

	if size%int64(PageSize) != 0 {
		return fmt.Errorf(" invalid size for a page blob. The size of the file %v (%v) is not a multiple of %v ", source.SourceName, source.Size, PageSize)
	}

	if size > int64(maxPageBlobSize) {
		return fmt.Errorf(" the file %v is too big (%v). Tha maximum size of a page blob is %v ", source.SourceName, source.Size, maxPageBlobSize)
	}

	if blockSize > maxPageSize || blockSize < PageSize {
		return fmt.Errorf(" invalid block size for page blob: %v. The value must be greater than %v and less than %v", PageSize, maxPageSize)
	}

	//if the max retries is exceeded, panic will happen, hence no error is returned.
	headers := make(map[string]string)
	userAgent, _ := util.GetUserAgentInfo()
	headers["User-Agent"] = userAgent
	util.RetriableOperation(func(r int) error {
		if err := (*t.StorageClient).PutPageBlob(t.Container, (*source).TargetAlias, size, headers); err != nil {
			t.resetClient()
			return err
		}
		return nil
	})

	return nil
}

//CommitList implements CommitList from the pipeline.TargetPipeline interface.
//Passthrough no need to a commit for page blob.
func (t *AzurePage) CommitList(listInfo *pipeline.TargetCommittedListInfo, NumberOfBlocks int, targetName string) (msg string, err error) {

	msg = "Page blob committed"
	err = nil
	return
}

//ProcessWrittenPart implements ProcessWrittenPart from the pipeline.TargetPipeline interface.
//Passthrough no need to process a written part when transferring to a page blob.
func (t *AzurePage) ProcessWrittenPart(result *pipeline.WorkerResult, listInfo *pipeline.TargetCommittedListInfo) (requeue bool, err error) {
	requeue = false
	err = nil
	return
}

//WritePart implements WritePart from the pipeline.TargetPipeline interface.
//Performs a PUT page operation with the data contained in the part.
//This assumes the part.BytesToRead is a multiple of the PageSize
func (t *AzurePage) WritePart(part *pipeline.Part) (duration time.Duration, startTime time.Time, numOfRetries int, err error) {

	offset := int64(part.Offset)
	endByte := int64(part.Offset + uint64(part.BytesToRead) - 1)
	headers := make(map[string]string)
	//if the max retries is exceeded, panic will happen, hence no error is returned.
	duration, startTime, numOfRetries = util.RetriableOperation(func(r int) error {
		//computation of the MD5 happens is done by the readers.
		if part.IsMD5Computed() {
			headers["Content-MD5"] = part.MD5()
		}
		userAgent, _ := util.GetUserAgentInfo()
		headers["User-Agent"] = userAgent

		if err := (*t.StorageClient).PutPage(t.Container, part.TargetAlias, offset, endByte, "update", part.Data, headers); err != nil {
			util.PrintfIfDebug("WritePart -> |%v|%v|%v|%v", part.Offset, len(part.Data), part.TargetAlias, err)
			t.resetClient()
			return err
		}

		util.PrintfIfDebug("WritePart -> |%v|%v|%v", part.Offset, len(part.Data), part.TargetAlias)

		return nil
	})

	return
}

func (t *AzurePage) resetClient() {
	client := util.GetBlobStorageClientWithNewHTTPClient(t.Creds.AccountName, t.Creds.AccountKey)
	t.StorageClient = &client
}
