package setcontext

import (
	"fmt"

	"github.com/danielfoehrkn/kubectlSwitch/pkg"
	"github.com/danielfoehrkn/kubectlSwitch/pkg/store"
	"github.com/danielfoehrkn/kubectlSwitch/pkg/subcommands/history/util"
	kubeconfigutil "github.com/danielfoehrkn/kubectlSwitch/pkg/util/kubectx_copied"
	"github.com/danielfoehrkn/kubectlSwitch/types"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func SetContext(desiredContext string, stores []store.KubeconfigStore, switchConfig *types.Config, stateDir string) error {
	c, err := pkg.DoSearch(stores, switchConfig, stateDir)
	if err != nil {
		return err
	}

	var mError *multierror.Error
	for discoveredContext := range *c {
		if discoveredContext.Error != nil {
			// remember in case the wanted context name cannot be found
			mError = multierror.Append(mError, discoveredContext.Error)
			continue
		}

		if discoveredContext.Store == nil {
			// this should not happen
			logger.Debugf("store returned from search is nil. This should not happen")
			continue
		}

		kubeconfigStore := *discoveredContext.Store
		contextWithoutFolderPrefix := kubeconfigutil.GetContextWithoutFolderPrefix(discoveredContext.Name)

		if desiredContext == discoveredContext.Name || desiredContext == contextWithoutFolderPrefix || desiredContext == discoveredContext.Alias {
			kubeconfigData, err := kubeconfigStore.GetKubeconfigForPath(discoveredContext.Path)
			if err != nil {
				return err
			}

			kubeconfig, err := kubeconfigutil.ParseTemporaryKubeconfig(kubeconfigData)
			if err != nil {
				return fmt.Errorf("failed to parse kubeconfig: %v", err)
			}

			originalContextBeforeAlias := ""
			if len(discoveredContext.Alias) > 0 {
				originalContextBeforeAlias = contextWithoutFolderPrefix
			}

			if err := kubeconfig.SetContext(desiredContext, originalContextBeforeAlias); err != nil {
				return err
			}

			tempKubeconfigPath, err := kubeconfig.WriteTemporaryKubeconfigFile()
			if err != nil {
				return fmt.Errorf("failed to write temporary kubeconfig file: %v", err)
			}

			if err := util.AppendContextToHistory(desiredContext); err != nil {
				logger.Warnf("failed to append context to history file: %v", err)
			}

			// print kubeconfig path to std.out -> captured by calling bash script to set KUBECONFIG environment Variable
			fmt.Print(tempKubeconfigPath)
			return nil
		}
	}

	if mError != nil {
		return fmt.Errorf("context with name %q not found. Possibly due to errors: %v", desiredContext, mError.Error())
	}

	return fmt.Errorf("context with name %q not found", desiredContext)
}
