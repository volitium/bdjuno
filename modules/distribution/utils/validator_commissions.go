package utils

import (
	"context"
	"fmt"

	"github.com/desmos-labs/juno/client"

	"github.com/forbole/bdjuno/database"
	"github.com/forbole/bdjuno/types"

	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/rs/zerolog/log"

	"github.com/forbole/bdjuno/types/config"
)

// UpdateValidatorsCommissionAmounts updates the validator commission amounts
func UpdateValidatorsCommissionAmounts(cfg *config.Config, height int64, client distrtypes.QueryClient, db *database.Db) {
	interval := cfg.GetDistributionConfig().GetDistributionFrequency()
	if interval == 0 {
		log.Debug().Str("module", "distribution").Msg("validator commission refresh interval set to 0. Skipping refresh")
		return
	}

	hasCommission, error := db.HasValidatorCommission()
	if error != nil {
		log.Error().Str("module", "distribution").Err(error).Int64("height", height).
			Msg("error while checking validator commission amount")
	}

	if !hasCommission || height%interval == 0 {
		go updateValidatorCommissions(height, client, db)
	}

}

// updateValidatorCommissions updates the validators commissions amounts
func updateValidatorCommissions(height int64, client distrtypes.QueryClient, db *database.Db) {
	log.Debug().Str("module", "distribution").
		Int64("height", height).
		Msg("updating validators commissions")

	validators, err := db.GetValidators()
	if err != nil {
		log.Error().Str("module", "distribution").Err(err).
			Int64("height", height).
			Msg("error while getting validators")
		return
	}

	if len(validators) == 0 {
		// No validators, just skip
		return
	}

	// Get all the commissions
	for _, validator := range validators {
		go updateValidatorCommission(height, client, validator, db)
	}
}

func updateValidatorCommission(height int64, distrClient distrtypes.QueryClient, validator types.Validator, db *database.Db) {
	commission, err := GetValidatorCommissionAmount(height, validator, distrClient)
	if err != nil {
		log.Error().Str("module", "distribution").Err(err).
			Int64("height", height).
			Str("validator", validator.GetOperator()).
			Send()
	}

	err = db.SaveValidatorCommissionAmount(commission)
	if err != nil {
		log.Error().Str("module", "distribution").Err(err).
			Int64("height", height).
			Str("validator", validator.GetOperator()).
			Msg("error while saving validator commission amounts")
	}
}

// GetValidatorCommissionAmount returns the amount of the validator commission for the given validator
func GetValidatorCommissionAmount(
	height int64, validator types.Validator, distrClient distrtypes.QueryClient,
) (types.ValidatorCommissionAmount, error) {
	res, err := distrClient.ValidatorCommission(
		context.Background(),
		&distrtypes.QueryValidatorCommissionRequest{ValidatorAddress: validator.GetOperator()},
		client.GetHeightRequestHeader(height),
	)
	if err != nil {
		return types.ValidatorCommissionAmount{}, fmt.Errorf("error while getting validator commission: %s", err)
	}

	return types.NewValidatorCommissionAmount(
		validator.GetOperator(),
		validator.GetSelfDelegateAddress(),
		res.Commission.Commission,
		height,
	), nil
}
