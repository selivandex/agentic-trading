<!-- @format -->

# Strategy Entity

Strategy entity represents trading strategies with configuration and performance tracking.

## Structure

```
strategy/
├── api/              # GraphQL queries and hooks
├── lib/              # Utility functions and formatters
├── model/            # TypeScript types
├── ui/               # React components
└── index.ts          # Public API
```

## Usage

```tsx
import { useUserStrategies, StrategyCard } from "@/entities/strategy";

function StrategiesList({ userID }: { userID: string }) {
  const { data, loading } = useUserStrategies(userID, "ACTIVE");

  if (loading) return <div>Loading...</div>;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {data?.userStrategies.edges.map(({ node: strategy }) => (
        <StrategyCard key={strategy.id} strategy={strategy} />
      ))}
    </div>
  );
}
```

## API

### Hooks

- `useStrategy(id)` - Get strategy by ID
- `useUserStrategies(userID, status?)` - Get user's strategies
- `useAllStrategies(limit?, offset?, status?)` - Get all strategies (admin)
- `useCreateStrategy()` - Create new strategy
- `useUpdateStrategy()` - Update strategy
- `usePauseStrategy()` - Pause strategy
- `useResumeStrategy()` - Resume strategy
- `useCloseStrategy()` - Close strategy

### Components

- `StrategyCard` - Display strategy information card
- `StrategyStatusBadge` - Display strategy status badge

### Formatters

- `formatStrategyStatus(status)` - Format status for display
- `getStrategyStatusColor(status)` - Get status color classes
- `formatMarketType(marketType)` - Format market type
- `formatRiskTolerance(riskTolerance)` - Format risk tolerance
- `formatRebalanceFrequency(frequency)` - Format rebalance frequency
- `formatCurrency(value, decimals?)` - Format currency value
- `formatPercentage(value, decimals?)` - Format percentage value
- `getPnLColorClass(value)` - Get color class for P&L
- `calculateROI(strategy)` - Calculate strategy ROI
