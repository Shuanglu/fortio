// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package configmap_test

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fortio.org/fortio/dflag"
	"fortio.org/fortio/dflag/configmap"
)

const (
	firstGoodDir  = "..9989_09_09_07_32_32.099817316"
	secondGoodDir = "..9289_09_10_03_32_32.039823124"
	badStaticDir  = "..1289_09_10_03_32_32.039823124"
)

var (
	assert  = dflag.Testify{}
	require = assert
	suite   = assert
)

type updaterTestSuite struct {
	dflag.TestSuite
	tempDir string

	flagSet   *flag.FlagSet
	staticInt *int
	dynInt    *dflag.DynInt64Value

	updater *configmap.Updater
}

func (s *updaterTestSuite) SetupTest() {
	var err error
	s.tempDir, err = ioutil.TempDir("/tmp", "updater_test")
	require.NoError(s.T(), err, "failed creating temp directory for testing")
	s.copyTestDataToDir()
	s.linkDataDirTo(firstGoodDir)

	s.flagSet = flag.NewFlagSet("updater_test", flag.ContinueOnError)
	s.dynInt = dflag.DynInt64(s.flagSet, "some_dynint", 1, "dynamic int for testing")
	s.staticInt = s.flagSet.Int("some_int", 1, "static int for testing")

	s.updater, err = configmap.New(s.flagSet, path.Join(s.tempDir, "testdata"))
	require.NoError(s.T(), err, "creating a config map must not fail")
}

// Tear down the updater.
func (s *updaterTestSuite) TearDownTest() {
	require.NoError(s.T(), os.RemoveAll(s.tempDir), "clearing up the test dir must not fail")
	_ = s.updater.Stop()
	time.Sleep(100 * time.Millisecond)
}

func (s *updaterTestSuite) copyTestDataToDir() {
	copyCmd := exec.Command("cp", "-a", "testdata", s.tempDir)
	require.NoError(s.T(), copyCmd.Run(), "copying testdata directory to tempdir must not fail")
	// We are storing file testdata/9989_09_09_07_32_32.099817316 and renaming it to testdata/..9989_09_09_07_32_32.099817316,
	// because go modules don't allow repos with files with .. in their filename. See https://github.com/golang/go/issues/27299.
	for _, p := range []string{firstGoodDir, secondGoodDir, badStaticDir} {
		pOld := filepath.Join(s.tempDir, "testdata", strings.TrimPrefix(p, ".."))
		pNew := filepath.Join(s.tempDir, "testdata", p)
		require.NoError(s.T(), os.Rename(pOld, pNew), "renaming %q to %q failed", pOld, pNew)
	}
}

func (s *updaterTestSuite) linkDataDirTo(newDataDir string) {
	copyCmd := exec.Command("ln", "-s", "-n", "-f",
		path.Join(s.tempDir, "testdata", newDataDir),
		path.Join(s.tempDir, "testdata", "..data"))
	require.NoError(s.T(), copyCmd.Run(), "relinking ..data in tempdir tempdir must not fail")
}

func (s *updaterTestSuite) TestInitializeFailsOnBadFormedFlag() {
	s.linkDataDirTo(badStaticDir)
	require.Error(s.T(), s.updater.Initialize(), "the updater initialize should return error on bad flags")
}

func (s *updaterTestSuite) TestSetupFunction() {
	tmpU, err := configmap.Setup(s.flagSet, path.Join(s.tempDir, "testdata"))
	require.NoError(s.T(), err, "setup for a config map must not fail")
	require.Error(s.T(), tmpU.Initialize(), "should error with already started")
	require.Error(s.T(), tmpU.Start(), "should error with already started")
	require.NoError(s.T(), tmpU.Stop(), "stopping the watcher should succeed")
}

func (s *updaterTestSuite) TestInitializeSetsValues() {
	require.NoError(s.T(), s.updater.Initialize(), "the updater initialize should not return errors on good flags")
	assert.EqualValues(s.T(), *s.staticInt, 1234, "staticInt should be some_int from first directory")
	assert.EqualValues(s.T(), s.dynInt.Get(), int64(10001), "staticInt should be some_int from first directory")
}

func (s *updaterTestSuite) TestDynamicUpdatesPropagate() {
	require.NoError(s.T(), s.updater.Initialize(), "the updater initialize should not return errors on good flags")
	require.NoError(s.T(), s.updater.Start(), "updater start should not return an error")
	s.linkDataDirTo(secondGoodDir)
	eventually(s.T(), 1*time.Second,
		assert.ObjectsAreEqualValues, int64(20002),
		func() interface{} { return s.dynInt.Get() },
		"some_dynint value should change to the value from secondGoodDir")
}

func TestUpdaterSuite(t *testing.T) {
	suite.Run(t, &updaterTestSuite{})
}

type (
	assertFunc func(expected, actual interface{}) bool
	getter     func() interface{}
)

// eventually tries a given Assert function 5 times over the period of time.
func eventually(t *testing.T, duration time.Duration,
	af assertFunc, expected interface{}, actual getter, msgFmt string, msgArgs ...interface{},
) {
	increment := duration / 5
	for i := 0; i < 5; i++ {
		time.Sleep(increment)
		if af(expected, actual()) {
			return
		}
	}
	t.Fatalf(msgFmt, msgArgs...)
}
