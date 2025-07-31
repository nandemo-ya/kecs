package tui2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui2"
)

var _ = Describe("CommandPalette", func() {
	var (
		model *tui2.Model
	)

	BeforeEach(func() {
		m := tui2.NewModel()
		model = &m
	})

	Describe("Command Filtering", func() {
		It("should show all commands when query is empty", func() {
			cp := model.GetCommandPalette()
			cp.FilterCommands("", model)
			Expect(len(cp.GetFilteredCommands())).To(BeNumerically(">", 0))
		})

		It("should filter commands by name", func() {
			cp := model.GetCommandPalette()
			cp.FilterCommands("help", model)
			found := false
			for _, cmd := range cp.GetFilteredCommands() {
				if cmd.Name == "help" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should filter commands by alias", func() {
			cp := model.GetCommandPalette()
			cp.FilterCommands("h", model)
			found := false
			for _, cmd := range cp.GetFilteredCommands() {
				if cmd.Name == "help" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should only show available commands", func() {
			cp := model.GetCommandPalette()
			
			// When no instance is selected, cluster commands should not be available
			model.SetSelectedInstance("")
			cp.FilterCommands("goto clusters", model)
			Expect(len(cp.GetFilteredCommands())).To(Equal(0))

			// When instance is selected, cluster commands should be available
			model.SetSelectedInstance("test-instance")
			cp.FilterCommands("goto clusters", model)
			Expect(len(cp.GetFilteredCommands())).To(BeNumerically(">", 0))
		})
	})

	Describe("Command Execution", func() {
		It("should execute commands by name", func() {
			cp := model.GetCommandPalette()
			result, err := cp.ExecuteByName("help", model)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Showing help"))
			Expect(model.IsHelpShown()).To(BeTrue())
		})

		It("should execute commands by alias", func() {
			cp := model.GetCommandPalette()
			result, err := cp.ExecuteByName("?", model)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Showing help"))
			Expect(model.IsHelpShown()).To(BeTrue())
		})

		It("should return error for unknown commands", func() {
			cp := model.GetCommandPalette()
			_, err := cp.ExecuteByName("unknown-command", model)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown command"))
		})

		It("should add executed commands to history", func() {
			cp := model.GetCommandPalette()
			_, _ = cp.ExecuteByName("help", model)
			history := cp.PreviousFromHistory()
			Expect(history).To(Equal("help"))
		})
	})

	Describe("History Navigation", func() {
		It("should navigate through command history", func() {
			cp := model.GetCommandPalette()
			
			// Execute some commands to build history
			_, _ = cp.ExecuteByName("help", model)
			_, _ = cp.ExecuteByName("refresh", model)
			_, _ = cp.ExecuteByName("search", model)

			// Navigate backwards
			Expect(cp.PreviousFromHistory()).To(Equal("search"))
			Expect(cp.PreviousFromHistory()).To(Equal("refresh"))
			Expect(cp.PreviousFromHistory()).To(Equal("help"))

			// Navigate forwards
			Expect(cp.NextFromHistory()).To(Equal("refresh"))
			Expect(cp.NextFromHistory()).To(Equal("search"))
			Expect(cp.NextFromHistory()).To(Equal(""))
		})

		It("should not duplicate commands in history", func() {
			cp := model.GetCommandPalette()
			
			_, _ = cp.ExecuteByName("help", model)
			_, _ = cp.ExecuteByName("help", model)
			
			Expect(cp.PreviousFromHistory()).To(Equal("help"))
			Expect(cp.PreviousFromHistory()).To(Equal("help")) // Should stay at the same position
		})
	})

	Describe("Result Display", func() {
		It("should show results temporarily", func() {
			cp := model.GetCommandPalette()
			
			_, _ = cp.ExecuteByName("help", model)
			
			Expect(cp.IsShowingResult()).To(BeTrue())
			Expect(cp.GetLastResult()).To(Equal("Showing help"))
			Expect(cp.ShouldShowResult()).To(BeTrue())
		})
	})
})