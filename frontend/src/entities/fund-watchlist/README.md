# Fund Watchlist Entity

Fund watchlist entity represents trading symbols being monitored by the fund.

## Structure

```
fund-watchlist/
├── api/              # GraphQL queries and hooks
├── lib/              # Utility functions, formatters, CRUD config
├── model/            # TypeScript types
├── ui/               # React components
└── index.ts          # Public API
```

## Usage

### Basic Usage (Hooks)

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

### CRUD Interface

```tsx
import { FundWatchlistManager } from "@/entities/fund-watchlist";

function FundWatchlistPage() {
  return <FundWatchlistManager />;
}
```

The `FundWatchlistManager` component provides a full CRUD interface with:
- List view with search, sort, pagination
- Create/Edit/Delete operations
- Pause/Resume actions
- Batch operations (delete multiple items)
- Scopes: All, Active, Paused, Inactive
- Dynamic filters: Market Type, Category, Tier
- Relay-style pagination

## API

### Hooks

#### Query Hooks
- `useFundWatchlist(id)` - Get watchlist item by ID
- `useFundWatchlistBySymbol(symbol, marketType)` - Get item by symbol
- `useFundWatchlists(filters?)` - Get all watchlist items (legacy)
- `useMonitoredSymbols(marketType?)` - Get monitored symbols (legacy)

#### Mutation Hooks
- `useCreateFundWatchlist()` - Create watchlist item
- `useUpdateFundWatchlist()` - Update watchlist item
- `useDeleteFundWatchlist()` - Delete watchlist item
- `useToggleFundWatchlistPause()` - Toggle pause state
- `usePauseFundWatchlist()` - Pause watchlist item
- `useResumeFundWatchlist()` - Resume watchlist item
- `useBatchDeleteFundWatchlists()` - Batch delete watchlist items

### Components

- `WatchlistItemCard` - Display watchlist item card
- `FundWatchlistManager` - Full CRUD interface with actions

### CRUD Configuration

- `fundWatchlistCrudConfig` - CRUD configuration object for watchlist items

### Formatters

- `formatTier(tier)` - Format tier for display
- `getTierColor(tier)` - Get tier badge color
- `getWatchlistStatusColor(item)` - Get status color
- `getWatchlistStatusText(item)` - Get status text
- `formatMarketType(marketType)` - Format market type
- `formatCategory(category)` - Format category

## Scopes

The fund watchlist supports the following scopes (tabs):

- **All** - All watchlist items
- **Active** - Active and not paused items
- **Paused** - Paused items
- **Inactive** - Inactive items

## Filters

Dynamic filters available:

- **Market Type** - SPOT or FUTURES (select)
- **Category** - LARGE_CAP, MID_CAP, SMALL_CAP, DEFI, GAMING, AI, MEME (select)
- **Tier** - 1, 2, 3, 4 (multiselect)
