package testingtproxy_test

import (
	"os"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"

	"github.com/onsi/ginkgo/v2/internal"
	"github.com/onsi/ginkgo/v2/internal/testingtproxy"
	"github.com/onsi/ginkgo/v2/types"
)

type messagedCall struct {
	message    string
	callerSkip []int
}

var _ = Describe("Testingtproxy", func() {
	var t FullGinkgoTInterface

	var failFunc func(message string, callerSkip ...int)
	var skipFunc func(message string, callerSkip ...int)
	var reportFunc func() types.SpecReport

	var failFuncCall messagedCall
	var skipFuncCall messagedCall
	var offset int
	var reportToReturn types.SpecReport
	var buf *gbytes.Buffer
	var recoverCall bool

	var attachedProgressReporter func() string
	var attachProgressReporterCancelCalled bool

	BeforeEach(func() {
		recoverCall = false
		attachProgressReporterCancelCalled = false
		failFuncCall = messagedCall{}
		skipFuncCall = messagedCall{}
		offset = 3
		reportToReturn = types.SpecReport{}

		failFunc = func(message string, callerSkip ...int) {
			failFuncCall.message = message
			failFuncCall.callerSkip = callerSkip
		}

		skipFunc = func(message string, callerSkip ...int) {
			skipFuncCall.message = message
			skipFuncCall.callerSkip = callerSkip
		}

		reportFunc = func() types.SpecReport {
			return reportToReturn
		}
		ginkgoRecoverFunc := func() {
			recoverCall = true
		}

		attachProgressReporterFunc := func(f func() string) func() {
			attachedProgressReporter = f
			return func() {
				attachProgressReporterCancelCalled = true
			}
		}

		buf = gbytes.NewBuffer()

		t = testingtproxy.New(
			internal.NewWriter(buf),
			failFunc,
			skipFunc,
			DeferCleanup,
			reportFunc,
			AddReportEntry,
			ginkgoRecoverFunc,
			attachProgressReporterFunc,
			17,
			3,
			5,
			true,
			offset)
	})

	Describe("Cleanup", Ordered, func() {
		var didCleanupAfter bool
		It("supports cleanup", func() {
			Ω(didCleanupAfter).Should(BeFalse())
			t.Cleanup(func() {
				didCleanupAfter = true
			})
		})

		It("ran cleanup after the last test", func() {
			Ω(didCleanupAfter).Should(BeTrue())
		})
	})

	Describe("Setenv", func() {
		Context("when the environment variable does not exist", Ordered, func() {
			const key = "FLOOP_FLARP_WIBBLE_BLARP"

			BeforeAll(func() {
				os.Unsetenv(key)
			})

			It("sets the environment variable", func() {
				t.Setenv(key, "HELLO")
				Ω(os.Getenv(key)).Should(Equal("HELLO"))
			})

			It("cleans up after itself", func() {
				_, exists := os.LookupEnv(key)
				Ω(exists).Should(BeFalse())
			})
		})

		Context("when the environment variable does exist", Ordered, func() {
			const key = "FLOOP_FLARP_WIBBLE_BLARP"
			const originalValue = "HOLA"

			BeforeAll(func() {
				os.Setenv(key, originalValue)
			})

			It("sets it", func() {
				t.Setenv(key, "HELLO")
				Ω(os.Getenv(key)).Should(Equal("HELLO"))
			})

			It("cleans up after itself", func() {
				Ω(os.Getenv(key)).Should(Equal("HOLA"))
			})

			AfterAll(func() {
				os.Unsetenv(key)
			})
		})
	})

	Describe("TempDir", Ordered, func() {
		var tempDirA, tempDirB string

		It("creates temporary directories", func() {
			tempDirA = t.TempDir()
			tempDirB = t.TempDir()
			Ω(tempDirA).Should(BeADirectory())
			Ω(tempDirB).Should(BeADirectory())
			Ω(tempDirA).ShouldNot(Equal(tempDirB))
		})

		It("cleans up after itself", func() {
			Ω(tempDirA).ShouldNot(BeADirectory())
			Ω(tempDirB).ShouldNot(BeADirectory())
		})
	})

	It("supports Error", func() {
		t.Error("a", 17)
		Ω(failFuncCall.message).Should(Equal("a 17\n"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports Errorf", func() {
		t.Errorf("%s %d!", "a", 17)
		Ω(failFuncCall.message).Should(Equal("a 17!"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports Fail", func() {
		t.Fail()
		Ω(failFuncCall.message).Should(Equal("failed"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports FailNow", func() {
		t.Fail()
		Ω(failFuncCall.message).Should(Equal("failed"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports Fatal", func() {
		t.Fatal("a", 17)
		Ω(failFuncCall.message).Should(Equal("a 17\n"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports Fatalf", func() {
		t.Fatalf("%s %d!", "a", 17)
		Ω(failFuncCall.message).Should(Equal("a 17!"))
		Ω(failFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("ignores Helper", func() {
		cl := func() types.CodeLocation {
			GinkgoT().Helper()
			return types.NewCodeLocation(0)
		}() // this is the expected line
		_, fname, lnumber, _ := runtime.Caller(0)
		Ω(cl).Should(Equal(types.CodeLocation{
			FileName:   fname,
			LineNumber: lnumber - 1,
		}))
	})

	It("supports Log", func() {
		t.Log("a", 17)
		Ω(string(buf.Contents())).Should(Equal("  a 17\n"))
	})

	It("supports Logf", func() {
		t.Logf("%s %d!", "a", 17)
		Ω(string(buf.Contents())).Should(Equal("  a 17!\n"))
	})

	It("supports Name", func() {
		reportToReturn.ContainerHierarchyTexts = []string{"C.S."}
		reportToReturn.LeafNodeText = "Lewis"
		Ω(t.Name()).Should(Equal("C.S. Lewis"))
		Ω(GinkgoT().Name()).Should(ContainSubstring("supports Name"))
	})

	It("ignores Parallel", func() {
		GinkgoT().Parallel() //is a no-op
	})

	It("supports Skip", func() {
		t.Skip("a", 17)
		Ω(skipFuncCall.message).Should(Equal("a 17\n"))
		Ω(skipFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports SkipNow", func() {
		t.SkipNow()
		Ω(skipFuncCall.message).Should(Equal("skip"))
		Ω(skipFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("supports Skipf", func() {
		t.Skipf("%s %d!", "a", 17)
		Ω(skipFuncCall.message).Should(Equal("a 17!"))
		Ω(skipFuncCall.callerSkip).Should(Equal([]int{offset}))
	})

	It("returns the state of the test when asked if it was skipped", func() {
		reportToReturn.State = types.SpecStatePassed
		Ω(t.Skipped()).Should(BeFalse())
		reportToReturn.State = types.SpecStateSkipped
		Ω(t.Skipped()).Should(BeTrue())
	})

	It("can add report entries with visibility Always", func() {
		cl := types.NewCodeLocation(0)
		t.AddReportEntryVisibilityAlways("hey", 3)
		entry := CurrentSpecReport().ReportEntries[0]
		Ω(entry.Visibility).Should(Equal(types.ReportEntryVisibilityAlways))
		Ω(entry.Name).Should(Equal("hey"))
		Ω(entry.GetRawValue()).Should(Equal(3))
		Ω(entry.Location.FileName).Should(Equal(cl.FileName))
		Ω(entry.Location.LineNumber).Should(Equal(cl.LineNumber + 1))
	})

	It("can add report entries with visibility FailureOrVerbose", func() {
		cl := types.NewCodeLocation(0)
		t.AddReportEntryVisibilityFailureOrVerbose("hey", 3)
		entry := CurrentSpecReport().ReportEntries[0]
		Ω(entry.Visibility).Should(Equal(types.ReportEntryVisibilityFailureOrVerbose))
		Ω(entry.Name).Should(Equal("hey"))
		Ω(entry.GetRawValue()).Should(Equal(3))
		Ω(entry.Location.FileName).Should(Equal(cl.FileName))
		Ω(entry.Location.LineNumber).Should(Equal(cl.LineNumber + 1))
	})

	It("can add report entries with visibility Never", func() {
		cl := types.NewCodeLocation(0)
		t.AddReportEntryVisibilityNever("hey", 3)
		entry := CurrentSpecReport().ReportEntries[0]
		Ω(entry.Visibility).Should(Equal(types.ReportEntryVisibilityNever))
		Ω(entry.Name).Should(Equal("hey"))
		Ω(entry.GetRawValue()).Should(Equal(3))
		Ω(entry.Location.FileName).Should(Equal(cl.FileName))
		Ω(entry.Location.LineNumber).Should(Equal(cl.LineNumber + 1))
	})

	It("can print to the GinkgoWriter", func() {
		t.Print("hi", 3)
		Ω(string(buf.Contents())).Should(Equal("  hi3"))
	})

	It("can printf to the GinkgoWriter", func() {
		t.Printf("hi %d", 3)
		Ω(string(buf.Contents())).Should(Equal("  hi 3"))
	})

	It("can println to the GinkgoWriter", func() {
		t.Println("hi", 3)
		Ω(string(buf.Contents())).Should(Equal("  hi 3\n"))
	})

	It("can provides a correctly configured Ginkgo Formatter", func() {
		Ω(t.F("{{blue}}%d{{/}}", 3)).Should(Equal("3"))
	})

	It("can printf to the GinkgoWriter", func() {
		Ω(t.Fi(1, "{{blue}}%d{{/}}", 3)).Should(Equal("  3"))
	})

	It("can println to the GinkgoWriter", func() {
		Ω(t.Fiw(1, 5, "{{blue}}%d{{/}} a number", 3)).Should(Equal("  3 a\n  number"))
	})

	It("can provide GinkgoRecover", func() {
		Ω(recoverCall).Should(BeFalse())
		t.GinkgoRecover()
		Ω(recoverCall).Should(BeTrue())
	})

	Describe("DeferCleanup", Ordered, func() {
		var a int
		It("provides access to DeferCleanup", func() {
			a = 3
			t.DeferCleanup(func(newA int) {
				a = newA
			}, 4)
		})

		It("provides access to DeferCleanup", func() {
			Ω(a).Should(Equal(4))
		})
	})

	It("provides the random seed", func() {
		Ω(t.RandomSeed()).Should(Equal(int64(17)))
	})

	It("provides the parallel process", func() {
		Ω(t.ParallelProcess()).Should(Equal(3))
	})

	It("provides the parallel total", func() {
		Ω(t.ParallelTotal()).Should(Equal(5))
	})

	It("can attach progress reports", func() {
		cancel := t.AttachProgressReporter(func() string {
			return "my report"
		})
		Ω(attachedProgressReporter()).Should(Equal("my report"))
		Ω(attachProgressReporterCancelCalled).Should(BeFalse())
		cancel()
		Ω(attachProgressReporterCancelCalled).Should(BeTrue())
	})

})
