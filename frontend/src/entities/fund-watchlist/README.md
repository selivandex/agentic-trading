# Fund Watchlist Entity

Fund watchlist entity represents trading symbols being monitored by the fund.

## Structure

```
fund-watchlist/
├── api/              # GraphQL queries and hooks
├── lib/              # Utility functions and formatters
├── model/            # TypeScript types
├── ui/               # React components
└── index.ts          # Public API
```

## Usage

```tsx
import { useMonitoredSymbols, WatchlistItemCard } from "@/entities/fund-watchlist";

function WatchlistGrid() {
  const { data, loading } = useMonitoredSymbols("FUTURES");
  
  if (loading) return <div>Loading...</div>;
  
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {data?.monitoredSymbols.map((item) => (
        <WatchlistItemCard key={item.id} item={item} />
      ))}
    </div>
  );
}
```

## API

### Hooks

- `useFundWatchlist(id)` - Get watchlist item by ID
- `useFundWatchlistBySymbol(symbol, marketType)` - Get item by symbol
- `useFundWatchlists(filters?)` - Get all watchlist items
- `useMonitoredSymbols(marketType?)` - Get monitored symbols
- `useCreateFundWatchlist()` - Create watchlist item
- `useUpdateFundWatchlist()` - Update watchlist item
- `useDeleteFundWatchlist()` - Delete watchlist item
- `useToggleFundWatchlistPause()` - Toggle pause state

### Components

- `WatchlistItemCard` - Display watchlist item card

### Formatters

- `formatTier(tier)` - Format tier for display
- `getTierColor(tier)` - Get tier badge color
- `getWatchlistStatusColor(item)` - Get status color
- `getWatchlistStatusText(item)` - Get status text
- `formatMarketType(marketType)` - Format market type
- `formatCategory(category)` - Format category


