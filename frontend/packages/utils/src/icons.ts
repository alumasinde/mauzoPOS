// Category/icon helpers shared by the back office and the POS till.
// Maps a display name (category, tile) to a key in @mauzo/ui's Icon set so
// category chips and POS tiles get a real icon instead of an emoji.

const RULES: [RegExp, string][] = [
  [/baker|bread|bun|pastry|cake/i, 'loaf'],
  [/butch|meat|beef|goat|pork|chicken|fish|offal|poultry/i, 'drumstick'],
  [/dairy|milk|yoghurt|cheese|egg/i, 'bottle'],
  [/beverage|drink|soda|juice|water|beer|wine|spirit|liquor|soft/i, 'cup'],
  [/snack|crisp|biscuit|cookie|confection|sweet|candy/i, 'cookie'],
  [/produce|fruit|vegetable|fresh|salad/i, 'carrot'],
  [/grocery|grocer|food|cereal|rice|flour|grain|staple/i, 'basket'],
  [/frozen/i, 'snowflake'],
  [/pharma|medicine|drug|prescri|otc|supplement|vitamin/i, 'pill'],
  [/personal|care|cosmetic|toilet|beauty|hygiene/i, 'sparkle'],
  [/household|clean|deterg|laundry|soap/i, 'spray'],
  [/baby|infant|diaper/i, 'baby'],
  [/pet|animal/i, 'paw'],
  [/cloth|apparel|fashion|wear|shoe/i, 'shirt'],
  [/electronic|phone|accessor|applian|computer|airtime/i, 'monitor'],
  [/hardware|tool|plumb|electrical|building/i, 'wrench'],
  [/stationery|book|school|office/i, 'book'],
  [/kitchen|utensil|cook|homeware/i, 'utensils'],
  [/main|breakfast|fast|dessert|menu|meal|food/i, 'utensils'],
  [/liquor|bar|pub/i, 'wine'],
];

// Stable fallback: spread unmatched names across a neutral set so tiles do
// not all look identical, and the same name always gets the same icon.
const FALLBACKS = ['tag', 'box', 'basket', 'bottle', 'cart', 'tags', 'gift', 'box'];

/** Icon name for a category/product-group display name. */
export function categoryIcon(name: string): string {
  for (const [re, icon] of RULES) if (re.test(name)) return icon;
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return FALLBACKS[h % FALLBACKS.length] ?? 'tag';
}

/** Icon name for a payment method code (cash, card, mpesa, …). */
export function methodIcon(code: string): string {
  switch (code) {
    case 'cash':
      return 'banknote';
    case 'card':
      return 'credit-card';
    case 'bank':
      return 'landmark';
    case 'credit':
      return 'book';
    case 'mpesa':
    case 'mobile':
      return 'phone';
    default:
      return 'wallet';
  }
}
