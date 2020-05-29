package main

import (
    "context"
    "encoding/json"
    "flag"
    "io/ioutil"
    "log"
    "os"
    "net/http"

    "github.com/multiformats/go-multiaddr"

    "github.com/Multi-Tier-Cloud/common/util"
    "github.com/Multi-Tier-Cloud/common/p2pnode"
    "github.com/Multi-Tier-Cloud/service-manager/conf"
    "github.com/Multi-Tier-Cloud/service-manager/lca"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultKeyFile = "~/.privKeyAlloc"

func init() {
    // Set up logging defaults
    log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
}

func main () {
    var err error

    // Parse options
    configPath := flag.String("configfile", "../conf/conf.json", "path to config file to use")
    var keyFlags util.KeyFlags
    var bootstraps *[]multiaddr.Multiaddr
    if keyFlags, err = util.AddKeyFlags(defaultKeyFile); err != nil {
        log.Fatalln(err)
    }
    if bootstraps, err = util.AddBootstrapFlags(); err != nil {
        log.Fatalln(err)
    }
    flag.Parse()

    priv, err := util.CreateOrLoadKey(keyFlags)
    if err != nil {
        log.Fatalln(err)
    }

    // Start Prometheus endpoint for stats collection
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":9101", nil)

    ctx := context.Background()

    // Read in config file
    config := conf.Config{}
    configFile, err := os.Open(*configPath)
    if err != nil {
        panic(err)
    }
    configByte, err := ioutil.ReadAll(configFile)
    if err != nil {
        configFile.Close()
        panic(err)
    }
    err = json.Unmarshal(configByte, &config)
    if err != nil {
        configFile.Close()
        panic(err)
    }
    configFile.Close()

    // If CLI didn't specify any bootstraps, fallback to configuration file
    if len(*bootstraps) == 0 {
        if len(config.Bootstraps) == 0 {
            log.Fatalln("ERROR: Must specify at least one bootstrap node" +
                "through a command line flag or the configuration file")
        }

        *bootstraps, err = p2pnode.StringsToMultiaddrs(config.Bootstraps)
        if err != nil {
            log.Fatalln(err)
        }
    }

    // Spawn LCA Allocator
    log.Println("Spawning LCA Allocator")
    _, err = lca.NewLCAAllocator(ctx, *bootstraps, priv)
    if err != nil {
        panic(err)
    }

    // Wait for connection
    log.Println("Waiting for requests...")
    select {}
}
