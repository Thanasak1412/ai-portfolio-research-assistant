export const maximumAssetSearchCodePoints = 100;

export function assetSearchValidationMessage(value: string): string | null {
  return Array.from(value).length <= maximumAssetSearchCodePoints
    ? null
    : "Search must be 100 characters or fewer.";
}
