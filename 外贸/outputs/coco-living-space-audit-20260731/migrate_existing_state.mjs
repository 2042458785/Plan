import fs from "node:fs/promises";
import path from "node:path";

const oldDir = path.resolve("outputs/coco-living-space-20260730");
const runDir = path.resolve("outputs/coco-living-space-audit-20260731");

const ndjson = await fs.readFile(path.join(oldDir, "COCO_Living_Space_高销量商品与1688搜货表.xlsx.inspect.ndjson"), "utf8");
const table = ndjson.split(/\r?\n/).filter(Boolean).map((line) => JSON.parse(line))
  .find((entry) => entry.kind === "table" && entry.sheet === "全部商品" && entry.address === "A1:Z889");
if (!table) throw new Error("未找到全部商品 A1:Z889 表格证据");

const imageIndex = JSON.parse(await fs.readFile(path.join(oldDir, "product_image_index.json"), "utf8"));
const oldProgress = JSON.parse(await fs.readFile(path.join(oldDir, "shopee_progress_results.json"), "utf8"));
const imageById = new Map(imageIndex.records.map((record) => [String(record.itemId), record]));
const rows = table.values.slice(4);
const collectedAt = "2026-07-30T00:00:00+08:00";

const catalogProducts = rows.map((row) => {
  const itemId = String(row[4]);
  const image = imageById.get(itemId);
  const monthlyRaw = row[8] || null;
  return {
    itemId,
    title: row[5] || null,
    url: `https://shopee.com.my/product/115498741/${itemId}`,
    seoUrl: row[24] || null,
    imageUrl: image?.sourceUrl || null,
    imagePath: image?.filePath ? path.relative(runDir, image.filePath) : null,
    listingPriceRaw: row[6] == null ? null : `RM${Number(row[6]).toFixed(2)}`,
    listingPriceMin: row[6] == null ? null : Number(row[6]),
    listingMonthlySalesRaw: monthlyRaw,
    listingMonthlySalesLower: monthlyRaw ? Number(row[7]) : null,
    listingRating: row[9] == null ? null : Number(row[9]),
    sourcePage: row[2] == null ? null : Number(row[2]),
    sourceIndex: row[3] == null ? null : Number(row[3])
  };
});

const results = {};
for (const row of rows) {
  const itemId = String(row[4]);
  const hasTotal = Boolean(row[10]);
  const hasCart = row[13] != null && row[13] !== "";
  const hasReviews = Boolean(row[16] || row[17] || row[15]);
  if (!hasTotal && !hasCart && !hasReviews) continue;
  const raw = row[10] || null;
  const historicalCartNote = hasCart ? `旧工作簿记录数量10行总额RM${Number(row[13]).toFixed(2)}，但没有购物车恢复证据；仅作历史参考，不计入已核验价格。` : "";
  results[itemId] = {
    itemId,
    status: "partial",
    url: row[24] || `https://shopee.com.my/product/115498741/${itemId}`,
    verifiedAt: row[22] ? `${row[22]}T00:00:00+08:00` : collectedAt,
    totalSoldRaw: raw,
    totalSoldLower: row[11] == null ? null : Number(row[11]),
    totalSoldExact: raw ? !/k\+/i.test(raw) : null,
    detailPriceRaw: row[12] || null,
    detailPrice: row[6] == null ? null : Number(row[6]),
    variantTested: null,
    quantityTested: null,
    cartLineTotal: null,
    cartUnitPrice: null,
    cartPriceBasis: null,
    ratingDisplayed: row[9] == null ? null : Number(row[9]),
    visibleReviewSampleCount: row[15] == null || row[15] === "" ? null : Number(row[15]),
    reviewCoverage: hasReviews ? "partial" : null,
    positiveSummary: row[16] || null,
    negativeSummary: row[17] || null,
    confidence: {
      pageFields: hasTotal ? "high" : "unknown",
      cartPrice: hasCart ? "medium" : "unknown",
      reviewThemes: hasReviews ? "medium" : "unknown",
      unseenPerformance: "unknown"
    },
    cartRestored: false,
    evidenceNote: `${row[23] || "旧工作簿迁移。"} ${historicalCartNote}`.trim()
  };
}

for (const [itemId, old] of Object.entries(oldProgress.results || {})) {
  const base = results[itemId] || { itemId, url: old.url };
  const cartRestored = /已移除测试商品|恢复|移除/.test(old.evidenceNote || "");
  const merged = {
    ...base,
    ...old,
    itemId,
    status: cartRestored && old.totalSoldRaw && old.cartLineTotal != null && old.visibleReviewSampleCount > 0 ? "verified" : "partial",
    verifiedAt: old.verifiedDate ? `${old.verifiedDate}T00:00:00+08:00` : base.verifiedAt,
    totalSoldExact: old.totalSoldRaw ? !/k\+/i.test(old.totalSoldRaw) : base.totalSoldExact,
    reviewCoverage: old.visibleReviewSampleCount > 0 ? "partial" : base.reviewCoverage,
    confidence: {
      pageFields: old.totalSoldRaw ? "high" : "unknown",
      cartPrice: cartRestored && old.cartLineTotal != null ? "high" : "medium",
      reviewThemes: old.visibleReviewSampleCount > 0 ? "medium" : "unknown",
      unseenPerformance: "unknown"
    },
    cartRestored,
    evidenceNote: old.evidenceNote || base.evidenceNote
  };
  if (!cartRestored) {
    merged.quantityTested = null;
    merged.cartLineTotal = null;
    merged.cartUnitPrice = null;
    merged.cartPriceBasis = null;
  }
  const buckets = [merged.fiveStarCount, merged.fourStarCount, merged.threeStarCount, merged.twoStarCount, merged.oneStarCount];
  if (!buckets.every((value) => Number.isFinite(value))) {
    merged.fiveStarCount = null;
    merged.fourStarCount = null;
    merged.threeStarCount = null;
    merged.twoStarCount = null;
    merged.oneStarCount = null;
  }
  results[itemId] = merged;
}

await fs.writeFile(path.join(runDir, "catalog.json"), JSON.stringify({ schemaVersion: 1, collectedAt, products: catalogProducts }, null, 2) + "\n");
await fs.writeFile(path.join(runDir, "progress.json"), JSON.stringify({ schemaVersion: 1, updatedAt: new Date().toISOString(), results }, null, 2) + "\n");

console.log(JSON.stringify({ catalog: catalogProducts.length, migratedResults: Object.keys(results).length }, null, 2));
