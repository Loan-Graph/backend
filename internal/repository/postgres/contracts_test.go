package postgres

import (
	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
)

var (
	_ lenderdomain.Repository   = (*LenderRepository)(nil)
	_ borrowerdomain.Repository = (*BorrowerRepository)(nil)
	_ loandomain.Repository     = (*LoanRepository)(nil)
	_ pooldomain.Repository     = (*PoolRepository)(nil)
	_ passportdomain.Repository = (*PassportRepository)(nil)
)
