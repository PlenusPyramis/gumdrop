package cmd

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/digitalocean/godo"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Configuration struct {
	ApiKey   string
	Droplets map[string]DropletConfig
}

type DropletConfig struct {
	Name       string
	Size       string
	Image      string
	Region     string
	FloatingIP string
	Volumes    []string
}

//Wrapper for survey.Ask that handles Ctrl-C SIGINT properly:
func ask(qs []*survey.Question, response interface{}, opts ...survey.AskOpt) {
	err := survey.Ask(qs, response, opts...)
	if err == terminal.InterruptErr {
		fmt.Println("interrupted")
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}
}

//Wrapper for survey.AskOne that handles Ctrl-C SIGINT properly:
func askOne(p survey.Prompt, response interface{}, opts ...survey.AskOpt) {
	err := survey.AskOne(p, response, opts...)
	if err == terminal.InterruptErr {
		fmt.Println("interrupted")
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}
}

func unmarshalConfig() (config Configuration) {
	config.Droplets = make(map[string]DropletConfig)
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func saveConfig(config Configuration) {
	//Custom save function for config to yaml.
	//Viper.WriteConfig is semi-broken and cannot filter items from configs.
	//See https://github.com/spf13/viper/issues/632

	configMap := viper.AllSettings()
	// Delete the config filename key to remove the self-reference to the output path:
	delete(configMap, "config")

	//Create yaml config string
	cfgYaml, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(viper.ConfigFileUsed())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.Write(cfgYaml)
	fmt.Println("Config file saved: ", viper.ConfigFileUsed())
}

func getDroplets(config Configuration) []string {
	var names = make([]string, 0)
	for _, cfg := range config.Droplets {
		names = append(names, cfg.Name)
	}
	sort.Strings(names)
	return names
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Droplet Configuration (list, create)",
}

var configCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new droplet configuration",
	//Override parent PersistentPreRun
	//Creates new config file if necessary
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfigFile()
	},
	Run: func(cmd *cobra.Command, args []string) {
		config := unmarshalConfig()
		client := GetClient(config.ApiKey)
		fmt.Println("Getting account info ... ")
		regions, regionMap, err := GetRegions(client)
		if err != nil {
			log.Fatal(err)
		}
		images, imagesMap, err := GetImages(client)
		if err != nil {
			log.Fatal(err)
		}
		_, sizesMap, err := GetSizes(client)
		if err != nil {
			log.Fatal(err)
		}
		answers := struct {
			Name       string
			Image      string
			Region     string
			Size       string
			FloatingIP string
			Volumes    []string
		}{}
		questions := []*survey.Question{
			{
				Name: "name",
				Prompt: &survey.Input{
					Message: "Choose a unique name for this droplet : ",
				},
			},
			{
				Name: "image",
				Prompt: &survey.Select{
					Message: "Choose the application image : ",
					Options: images,
				},
			},
			{
				Name: "region",
				Prompt: &survey.Select{
					Message: "Choose the region for the droplet :",
					Default: regions[0],
					Options: regions,
				},
			},
		}
		ask(questions, &answers)

		dropletSizes := make([]godo.Size, 0)
		for _, s := range sizesMap {
			//Filter out the legacy instance types that don't have dashes:
			if strings.Contains(s.Slug, "-") {
				//Filter out unavailble sizes:
				if s.Available {
					for _, region := range s.Regions {
						if strings.HasSuffix(answers.Region, region) {
							dropletSizes = append(dropletSizes, s)
						}
					}
				}
			}
		}
		if len(dropletSizes) == 0 {
			log.Fatal("Droplet sizes list is empty")
		}
		//Sort droplet sizes by price:
		sort.Slice(dropletSizes, func(i, j int) bool {
			return dropletSizes[i].PriceMonthly < dropletSizes[j].PriceMonthly
		})
		dropletSizeNames := make([]string, 0)
		for _, s := range dropletSizes {
			dropletSizeNames = append(dropletSizeNames, fmt.Sprintf("%s - $%0.2f/month", s.Slug, s.PriceMonthly))
		}
		questions = []*survey.Question{
			{
				Name: "size",
				Prompt: &survey.Select{
					Message: "Choose the droplet size : ",
					Default: dropletSizeNames[0],
					Options: dropletSizeNames,
				},
			},
		}
		ask(questions, &answers)

		useFloatingIP := true
		for {
			askOne(&survey.Confirm{
				Message: "Do you want to assign a floating IP address? ",
				Default: true,
			}, &useFloatingIP)
			if !useFloatingIP {
				break
			} else {
				// Ask to create a new IP or use an existing one:
				region := regionMap[answers.Region].Slug
				createIPmsg := fmt.Sprintf("Create a new Floating IP in the %s region", region)
				existingIPmsg := fmt.Sprintf("Use an existing Floating IP in the %s region", region)
				createIPAnswer := createIPmsg
				askOne(&survey.Select{
					Message: "Do you want to create a new Floating IP or use an existing one?",
					Options: []string{createIPmsg, existingIPmsg},
					Default: createIPmsg,
				}, &createIPAnswer)
				if createIPAnswer == createIPmsg {
					//Create new floating IP
					answers.FloatingIP = CreateFloatingIP(client, region)
					break
				} else {
					//Use existing floating IP
					floatingIPs, _, err := GetFloatingIPs(client, region)
					if err != nil {
						log.Fatal(err)
					}
					if len(floatingIPs) == 0 {
						fmt.Println(fmt.Sprintf("No existing floating IPs are available in the %s region", region))
						continue
					}
					askOne(&survey.Select{
						Message: "Choose an existing Floating IP address : ",
						Options: floatingIPs,
					}, &answers.FloatingIP)
					break
				}
			}
		}

		addVolumes := false
		for {
			askOne(&survey.Confirm{
				Message: "Do you want to assign more external volumes? ",
				Default: false,
			}, &addVolumes)
			if !addVolumes {
				break
			} else {
				// Ask to create a new volume or use an existing one:
				region := regionMap[answers.Region].Slug
				createVolumeMsg := fmt.Sprintf("Create a new volume in the %s region", region)
				existingVolumeMsg := fmt.Sprintf("Use an existing volume in the %s region", region)
				createVolumeAnswer := createVolumeMsg
				askOne(&survey.Select{
					Message: "Do you want to create a new volume or use an existing one?",
					Options: []string{createVolumeMsg, existingVolumeMsg},
					Default: createVolumeMsg,
				}, &createVolumeAnswer)
				if createVolumeAnswer == createVolumeMsg {
					//Create new volume
					name := ""
					askOne(&survey.Input{
						Message: "Enter the name for the new volume : ",
					}, &name)
					var size int64 = 1
					askOne(&survey.Input{
						Message: "Enter the size for the new volume (in GiB) : ",
					}, &size)
					answers.Volumes = append(answers.Volumes, CreateVolume(client, region, name, size))
					continue
				} else {
					//Use existing volume
					volumes, volumesMap, err := GetUnattachedVolumes(client, region)
					if err != nil {
						log.Fatal(err)
					}
					if len(volumes) == 0 {
						fmt.Println(fmt.Sprintf("No existing unattached volumes are available in the %s region", region))
						continue
					}
					volume := ""
					askOne(&survey.Select{
						Message: "Choose an existing/unattached volume : ",
						Options: volumes,
					}, &volume)
					answers.Volumes = append(answers.Volumes, volumesMap[volume].ID)
					continue
				}
			}
		}

		cfg := DropletConfig{
			Name:       answers.Name,
			Size:       sizesMap[answers.Size].Slug,
			Region:     regionMap[answers.Region].Slug,
			Image:      imagesMap[answers.Image].Slug,
			FloatingIP: strings.Split(answers.FloatingIP, " ")[0],
			Volumes:    answers.Volumes,
		}
		config.Droplets[cfg.Name] = cfg
		saveConfig(config)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing droplet configurations",
	Run: func(cmd *cobra.Command, args []string) {
		config := unmarshalConfig()
		client := GetClient(config.ApiKey)
		names := getDroplets(config)

		_, droplets, err := GetDroplets(client)
		if err != nil {
			log.Fatal(err)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Size", "Image", "Region", "Floating IP", "Status"})
		for _, name := range names {
			fmt.Println(name)
			dropCfg := config.Droplets[name]
			d := droplets[dropCfg.Name]
			status := "unused"
			normalStyle := tablewriter.Colors{}
			statusStyle := tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor}
			if d.ID != 0 && dropCfg.Region == d.Region.Slug {
				status = d.Status
				statusStyle = tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor}
			}
			table.Rich([]string{dropCfg.Name, dropCfg.Size, dropCfg.Image, dropCfg.Region, dropCfg.FloatingIP, status},
				[]tablewriter.Colors{normalStyle, normalStyle, normalStyle, normalStyle, normalStyle, statusStyle})
		}
		table.Render()
	},
}

func initConfigFile() {
	err := initConfig()
	if err != nil {
		if err.Error() == "No config file found" {
			cfgAbsPath, err := filepath.Abs(cfgFile)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Configuration file not found: ", cfgAbsPath)
			createNew := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Do you want to create a new config file (%s)?", cfgFile),
			}
			askOne(prompt, &createNew)
			if createNew {
				var apiKey = ""
				for {
					fmt.Println("\nCreate your Digital Ocean Personal Access Token (See https://cloud.digitalocean.com/account/api/tokens)")
					prompt := &survey.Input{Message: "Enter your Digital Ocean Personal Access Token :"}
					askOne(prompt, &apiKey, survey.WithValidator(survey.Required))
					client := GetClient(apiKey)
					_, err := GetAccount(client)
					if err == nil {
						break
					} else {
						fmt.Println(err)
					}
				}
				viper.Set("apikey", apiKey)
				viper.WriteConfigAs(cfgFile)
				if err := os.Chmod(cfgFile, 0600); err != nil {
					log.Fatal(err)
				}
				fmt.Println("\nCreated new config file: ", cfgFile)
			}
		} else {
			log.Fatal(err)
		}
		initConfig()
	}
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configCreateCmd)
	configCmd.AddCommand(configListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
