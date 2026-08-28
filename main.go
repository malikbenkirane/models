package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"models/litellm"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func run() error {
	rootCommand := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	opencodeCommand := &cobra.Command{
		Use: "opencode",
		RunE: func(cmd *cobra.Command, args []string) error {
			var models models = make(map[string]model)
			for _, m := range myModels {
				models[m.litellm] = model{
					Name:       m.name,
					Modalities: m.modalities,
					Attachment: m.attachment,
					Cost: cost{
						Input:      m.input,
						Output:     m.output,
						CacheRead:  m.cache,
						CacheWrite: m.cache,
					},
				}
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(models)
		},
	}
	litellmCommand := &cobra.Command{
		Use: "litellm",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			var b bytes.Buffer
			if err := litellm.ConfigJSON(&b, litellmModels(cfg)); err != nil {
				return err
			}
			{
				b, err := yaml.JSONToYAML(b.Bytes())
				if err != nil {
					return err
				}
				fmt.Println(string(b))
			}
			return nil
		},
	}
	pricingCommand := &cobra.Command{
		Use: "pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			sortKey, _ := cmd.Flags().GetString("sort")
			desc, _ := cmd.Flags().GetBool("desc")
			rows := append([]modelDescription(nil), myModels...)
			less := func(i, j int) bool {
				a, b := &rows[i], &rows[j]
				switch sortKey {
				case "output":
					return a.output.Less(b.output)
				case "cache":
					return a.cache.Less(b.cache)
				case "name":
					return a.name < b.name
				default:
					return a.input.Less(b.input)
				}
			}
			if desc {
				orig := less
				less = func(i, j int) bool { return orig(j, i) }
			}
			sort.SliceStable(rows, less)
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "MODEL\tLITELLM KEY\tPROVIDER\tINPUT $/Mtok\tOUTPUT $/Mtok\tCACHE $/Mtok")
			for _, m := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					m.name, m.litellm, m.provider, m.input, m.output, m.cache)
			}
			return tw.Flush()
		},
	}
	pricingCommand.Flags().String("sort", "input", "sort by column: input|output|cache|name")
	pricingCommand.Flags().Bool("desc", false, "sort descending")
	rootCommand.AddCommand(opencodeCommand, litellmCommand, pricingCommand)
	return rootCommand.Execute()
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
