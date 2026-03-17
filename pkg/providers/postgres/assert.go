package postgres

import "github.com/soloz-io/open-sbt/pkg/interfaces"

// Compile-time assertion: Storage must satisfy IStorage.
var _ interfaces.IStorage = (*Storage)(nil)
