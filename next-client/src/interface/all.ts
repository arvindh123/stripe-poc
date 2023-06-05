export interface TOrg {
    id: Number,
    name: String,
    email: String,
    stripe_id: String,
    stripe_sub: String,
    sub_status: String,
    plans: TPlan[]
}

export interface TProduct {
  id: string,
  active: boolean,
  name: string,
  description: string,
}

export interface TPlan {
  id: string,
  si_id: string,
  sub_id: string,
  active: boolean,
  quantity: number,
  amount: number,
  product: TProduct
}


export interface TPrice {
  active: boolean;
  billing_scheme: string;
  created: number;
  currency: string;
  currency_options?: null;
  custom_unit_amount?: null;
  deleted: boolean;
  id: string;
  livemode: boolean;
  lookup_key: string;
  metadata: TPriceMetadata;
  nickname: string;
  object: string;
  product: TProduct;
  recurring: TPriceRecurring;
  tax_behavior: string;
  tiers?: null;
  tiers_mode: string;
  transform_quantity?: null;
  type: string;
  unit_amount: number;
  unit_amount_decimal: string;
}
export interface TPriceMetadata {
  users_limit: string;
}
export interface TProduct {
  active: boolean;
  attributes?: (null)[] | null;
  caption: string;
  created: number;
  deactivate_on?: null;
  default_price: TProductDefaultPrice;
  deleted: boolean;
  description: string;
  id: string;
  images?: (null)[] | null;
  livemode: boolean;
  metadata: TProductMetadata;
  name: string;
  object: string;
  package_dimensions?: null;
  shippable: boolean;
  statement_descriptor: string;
  tax_code?: null;
  type: string;
  unit_label: string;
  updated: number;
  url: string;
}
export interface TProductDefaultPrice {
  active: boolean;
  billing_scheme: string;
  created: number;
  currency: string;
  currency_options?: null;
  custom_unit_amount?: null;
  deleted: boolean;
  id: string;
  livemode: boolean;
  lookup_key: string;
  metadata?: null;
  nickname: string;
  object: string;
  product?: null;
  recurring?: null;
  tax_behavior: string;
  tiers?: null;
  tiers_mode: string;
  transform_quantity?: null;
  type: string;
  unit_amount: number;
  unit_amount_decimal: string;
}
export interface TProductMetadata {
  users_limit: string;
}
export interface TPriceRecurring {
  aggregate_usage: string;
  interval: string;
  interval_count: number;
  trial_period_days: number;
  usage_type: string;
}
