package smoke

import (
	"github.com/rook/rook/e2e/framework/enums"
	"github.com/rook/rook/e2e/framework/manager"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

func TestBlockStorageSmokeSuite(t *testing.T) {
	suite.Run(t, new(BlockStorageTestSuite))
}

type BlockStorageTestSuite struct {
	suite.Suite
	rookPlatform enums.RookPlatformType
	k8sVersion   enums.K8sVersion
	rookTag      string
	helper       *SmokeTestHelper
}

func (suite *BlockStorageTestSuite) SetupTest() {
	var err error

	suite.rookPlatform, err = enums.GetRookPlatFormTypeFromString(env.Platform)

	require.Nil(suite.T(), err)

	suite.k8sVersion, err = enums.GetK8sVersionFromString(env.K8sVersion)

	require.Nil(suite.T(), err)

	suite.rookTag = env.RookTag

	require.NotEmpty(suite.T(), suite.rookTag, "RookTag parameter is required")

	err, rookInfra := rook_test_infra.GetRookTestInfraManager(suite.rookPlatform, true, suite.k8sVersion)

	require.Nil(suite.T(), err)

	skipRookInstall := env.SkipInstallRook == "true"

	rookInfra.ValidateAndSetupTestPlatform(skipRookInstall)

	err = rookInfra.InstallRook(suite.rookTag, skipRookInstall)

	require.Nil(suite.T(), err)

	suite.helper, err = CreateSmokeTestClient(rookInfra.GetRookPlatform())
	require.Nil(suite.T(), err)
}

func (suite *BlockStorageTestSuite) TestBlockStorage_SmokeTest() {

	suite.T().Log("Block Storage Smoke Test - Create,Mount,write to, read from  and Unmount Block")

	defer blockTestcleanup(suite.helper)
	rbc := suite.helper.GetBlockClient()
	suite.T().Log("Step 0 : Get Initial List Block")
	initBlockImages, _ := rbc.BlockList()

	suite.T().Log("step 1: Create block storage")
	_, cb_err := suite.helper.CreateBlockStorage()
	require.Nil(suite.T(), cb_err)
	require.True(suite.T(), retryBlockImageCountCheck(suite.helper, len(initBlockImages), 1), "Make sure a new block is created")
	suite.T().Log("Block Storage created successfully")

	suite.T().Log("step 2: Mount block storage")
	_, mt_err := suite.helper.MountBlockStorage()
	require.Nil(suite.T(), mt_err)
	suite.T().Log("Block Storage Mounted successfully")

	suite.T().Log("step 3: Write to block storage")
	_, wt_err := suite.helper.WriteToBlockStorage("Test Data", "testFile1")
	require.Nil(suite.T(), wt_err)
	suite.T().Log("Write to Block storage successfully")

	suite.T().Log("step 4: Read from  block storage")
	read, r_err := suite.helper.ReadFromBlockStorage("testFile1")
	require.Nil(suite.T(), r_err)
	require.Contains(suite.T(), read, "Test Data", "make sure content of the files is unchanged")
	suite.T().Log("Read from  Block storage successfully")

	suite.T().Log("step 5: Unmount block storage")
	_, unmt_err := suite.helper.UnMountBlockStorage()
	require.Nil(suite.T(), unmt_err)
	suite.T().Log("Block Storage unmounted successfully")

	suite.T().Log("step 6: Deleting block storage")
	_, db_err := suite.helper.DeleteBlockStorage()
	require.Nil(suite.T(), db_err)
	require.True(suite.T(), retryBlockImageCountCheck(suite.helper, len(initBlockImages), 0), "Make sure a new block is created")
	suite.T().Log("Block Storage deleted successfully")

}

func blockTestcleanup(h *SmokeTestHelper) {
	h.UnMountBlockStorage()
	h.DeleteBlockStorage()
	h.CleanUpDymanicBlockStorge()
}

// periodically checking if block image count has changed to expected value
// When creating pvc in k8s platform, it may take some time for the block Image to be bounded
func retryBlockImageCountCheck(h *SmokeTestHelper, imageCount int, expectedChange int) bool {
	inc := 0
	for inc < 10 {
		blockImages, _ := h.GetBlockClient().BlockList()
		if imageCount+expectedChange == len(blockImages) {
			return true
		} else {
			time.Sleep(time.Second * 2)
			inc++
		}
	}
	return false
}
