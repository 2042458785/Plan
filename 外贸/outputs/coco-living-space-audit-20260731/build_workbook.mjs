import fs from "node:fs/promises";
import path from "node:path";
import { FileBlob, SpreadsheetFile, Workbook } from "@oai/artifact-tool";

const runDir = path.resolve("outputs/coco-living-space-audit-20260731");
const outputPath = path.join(runDir, "COCO_Living_Space_Shopee全量审计_持续更新.xlsx");
const manifest = JSON.parse(await fs.readFile(path.join(runDir, "manifest.json"), "utf8"));
const catalog = JSON.parse(await fs.readFile(path.join(runDir, "catalog.json"), "utf8"));
const progress = JSON.parse(await fs.readFile(path.join(runDir, "progress.json"), "utf8"));
const results = progress.results || {};

const statuses = ["verified", "partial", "inaccessible", "verification_required", "blocked_cart_safety", "blocked_login", "pending"];
const statusLabel = {
  verified: "已核验",
  partial: "部分核验",
  inaccessible: "当前不可访问",
  verification_required: "需要验证码",
  blocked_cart_safety: "购物车安全阻塞",
  blocked_login: "登录阻塞",
  pending: "待核验"
};
const statusCounts = Object.fromEntries(statuses.map((s) => [s, 0]));
for (const product of catalog.products) statusCounts[results[product.itemId]?.status || "pending"]++;
const detailCount = Object.values(results).filter((r) => r.totalSoldRaw).length;
const cartCount = Object.values(results).filter((r) => r.cartRestored === true && r.cartLineTotal != null && r.quantityTested).length;
const reviewCount = Object.values(results).filter((r) => r.positiveSummary || r.negativeSummary).length;
const completeReviewCount = Object.values(results).filter((r) => r.reviewCoverage === "complete").length;

function compactConfidence(value) {
  if (!value) return "未知";
  if (typeof value === "string") return value;
  return `页面:${value.pageFields || "未知"}｜购物车:${value.cartPrice || "未知"}｜评论:${value.reviewThemes || "未知"}`;
}

function combined(product, index) {
  const r = results[product.itemId] || {};
  return {
    seq: index + 1,
    product,
    result: r,
    status: r.status || "pending",
    values: [
      index + 1, "", product.itemId, product.title || "", product.seoUrl || product.url || "",
      product.listingMonthlySalesRaw || "", product.listingMonthlySalesLower ?? null,
      product.listingPriceRaw || "", product.listingPriceMin ?? null, product.listingRating ?? null,
      r.totalSoldRaw || "", r.totalSoldLower ?? null, r.totalSoldExact ?? null,
      r.detailPriceRaw || "", r.detailPrice ?? null, r.variantTested || "",
      r.quantityTested ?? null, r.cartLineTotal ?? null, null,
      r.ratingDisplayed ?? null, r.ratingCount ?? null,
      r.fiveStarCount ?? null, r.fourStarCount ?? null, r.threeStarCount ?? null, r.twoStarCount ?? null, r.oneStarCount ?? null,
      r.visibleReviewSampleCount ?? null, r.reviewCoverage || "",
      r.positiveSummary || "", r.negativeSummary || "", statusLabel[r.status || "pending"],
      compactConfidence(r.confidence), r.evidenceNote || "", r.verifiedAt || "",
      product.imageUrl || "", product.imagePath || ""
    ]
  };
}

const joined = catalog.products.map(combined);
const highSales = joined.filter((x) => x.result.totalSoldLower >= 500).sort((a, b) => b.result.totalSoldLower - a.result.totalSoldLower);
const deep = joined.filter((x) => x.result.totalSoldRaw || x.result.cartLineTotal != null || x.result.positiveSummary || x.result.negativeSummary);

const wb = Workbook.create();
const overview = wb.worksheets.add("概览");
const all = wb.worksheets.add("全部商品");
const high = wb.worksheets.add("高销量商品");
const detail = wb.worksheets.add("深度核验");

const navy = "#203248";
const gold = "#B68A55";
const pale = "#F7F2EA";
const light = "#EAF0F6";
const line = "#D8DEE6";
const muted = "#657384";

overview.showGridLines = false;
overview.mergeCells("A1:H1");
overview.getRange("A1").values = [["COCO Living Space｜Shopee 全量店铺审计"]];
overview.getRange("A1:H1").format = { fill: navy, font: { bold: true, color: "#FFFFFF", fontSize: 18 }, verticalAlignment: "center" };
overview.getRange("A1:H1").format.rowHeight = 32;
overview.mergeCells("A2:H2");
overview.getRange("A2").values = [[`店铺：${manifest.shopName}｜区域：${manifest.region}｜清单采集：${catalog.collectedAt}｜本次状态更新：${progress.updatedAt}`]];
overview.getRange("A2:H2").format = { fill: light, font: { color: muted, fontSize: 10 }, verticalAlignment: "center" };
overview.getRange("A4:B12").values = [
  ["清单商品数", catalog.products.length],
  ["详情总销量有证据", detailCount],
  ["购物车价格安全核验", cartCount],
  ["评论已有摘要", reviewCount],
  ["评论完整翻页", completeReviewCount],
  ["已核验", statusCounts.verified],
  ["部分核验", statusCounts.partial],
  ["需要验证码", statusCounts.verification_required],
  ["仍待核验", statusCounts.pending]
];
overview.getRange("A4:A12").format = { fill: pale, font: { bold: true, color: navy }, borders: { preset: "inside", style: "thin", color: line } };
overview.getRange("B4:B12").format = { font: { bold: true, color: navy, fontSize: 14 }, numberFormat: "#,##0", horizontalAlignment: "center", borders: { preset: "inside", style: "thin", color: line } };
overview.mergeCells("D4:H4");
overview.getRange("D4").values = [["数据口径与置信度限制"]];
overview.getRange("D4:H4").format = { fill: gold, font: { bold: true, color: "#FFFFFF" } };
overview.mergeCells("D5:H12");
overview.getRange("D5").values = [[
  "1. 近月销量来自店铺列表；累计总销量只使用详情页文字，二者不混用。\n" +
  "2. ‘k+ Sold’只记下限，不当作精确销量。\n" +
  "3. 购物车价格只统计已确认测试商品被精确移除且购物车恢复的记录；旧表中缺少恢复证据的历史金额不再计为已核验。\n" +
  "4. 价格仅适用于被测变体、数量、账号和时间；未扣优惠券、运费、积分或结账优惠。\n" +
  "5. 评论摘要只代表可见样本；未翻完全部评论时均为部分覆盖。\n" +
  "6. 当前出现验证码时必须暂停，不能把未访问字段猜成零或下架。"
]];
overview.getRange("D5:H12").format = { fill: "#FFFFFF", font: { color: navy, fontSize: 10 }, wrapText: true, verticalAlignment: "top", borders: { preset: "outside", style: "thin", color: line } };
overview.getRange("A4:H12").format.rowHeight = 28;
overview.getRange("A1:H12").format.borders = { preset: "outside", style: "thin", color: line };
overview.getRange("A1:H12").format.columnWidth = 15;
overview.getRange("A1:A12").format.columnWidth = 22;
overview.getRange("D1:H12").format.columnWidth = 16;

const allHeaders = ["序号", "商品主图", "商品ID", "商品标题", "Shopee链接", "近月销量显示", "近月销量下限", "列表价格显示", "列表最低价(MYR)", "列表评分", "累计总销量显示", "累计总销量下限", "是否精确", "详情价格显示", "详情价(MYR)", "测试变体", "实际测试数量", "购物车行总额(MYR)", "实测单价(MYR)", "详情评分", "评分总数", "五星", "四星", "三星", "二星", "一星", "可见评论样本数", "评论覆盖", "好评主题", "差评/风险主题", "状态", "字段置信度", "证据备注", "核验时间", "图片来源URL", "本地主图路径"];

function setupDataSheet(sheet, title, note, rows, tableName, includeImages) {
  sheet.showGridLines = false;
  sheet.mergeCells("A1:AJ1");
  sheet.getRange("A1").values = [[title]];
  sheet.getRange("A1:AJ1").format = { fill: navy, font: { bold: true, color: "#FFFFFF", fontSize: 16 }, verticalAlignment: "center" };
  sheet.getRange("A1:AJ1").format.rowHeight = 30;
  sheet.mergeCells("A2:AJ2");
  sheet.getRange("A2").values = [[note]];
  sheet.getRange("A2:AJ2").format = { fill: light, font: { color: muted, fontSize: 9 }, wrapText: true, verticalAlignment: "center" };
  sheet.getRange("A4:AJ4").values = [allHeaders];
  const matrix = rows.map((x) => x.values);
  if (matrix.length) sheet.getRangeByIndexes(4, 0, matrix.length, allHeaders.length).values = matrix;
  const endRow = 4 + rows.length;
  if (rows.length) {
    for (let excelRow = 5; excelRow <= endRow; excelRow++) sheet.getRange(`S${excelRow}`).formulas = [[`=IF(OR(Q${excelRow}="",R${excelRow}=""),"",R${excelRow}/Q${excelRow})`]];
    const table = sheet.tables.add(`A4:AJ${endRow}`, true, tableName);
    table.style = "TableStyleMedium2";
    table.showFilterButton = true;
    sheet.getRange(`A5:AJ${endRow}`).format = { font: { color: navy, fontSize: 9 }, verticalAlignment: "center" };
    sheet.getRange(`F5:G${endRow}`).format.numberFormat = "#,##0";
    sheet.getRange(`I5:I${endRow}`).format.numberFormat = '"RM"#,##0.00';
    sheet.getRange(`L5:L${endRow}`).format.numberFormat = "#,##0";
    sheet.getRange(`O5:O${endRow}`).format.numberFormat = '"RM"#,##0.00';
    sheet.getRange(`Q5:Q${endRow}`).format.numberFormat = "#,##0";
    sheet.getRange(`R5:S${endRow}`).format.numberFormat = '"RM"#,##0.00';
    sheet.getRange(`U5:AA${endRow}`).format.numberFormat = "#,##0";
    sheet.getRange(`AH5:AH${endRow}`).format.numberFormat = "yyyy-mm-dd hh:mm";
    sheet.getRange(`D5:D${endRow}`).format.wrapText = true;
    sheet.getRange(`AC5:AG${endRow}`).format.wrapText = true;
    sheet.getRange(`AE5:AE${endRow}`).conditionalFormats.add("containsText", { text: "已核验", format: { fill: "#DDEFE5", font: { color: "#17633A", bold: true } } });
    sheet.getRange(`AE5:AE${endRow}`).conditionalFormats.add("containsText", { text: "需要验证码", format: { fill: "#FFF1CC", font: { color: "#8A5A00", bold: true } } });
    sheet.getRange(`AE5:AE${endRow}`).conditionalFormats.add("containsText", { text: "待核验", format: { fill: "#EEF1F4", font: { color: muted } } });
    sheet.getRange(`A5:AJ${endRow}`).format.rowHeight = includeImages ? 64 : 40;
  }
  sheet.getRange("A4:AJ4").format = { fill: gold, font: { bold: true, color: "#FFFFFF", fontSize: 9 }, wrapText: true, verticalAlignment: "center" };
  sheet.getRange("A4:AJ4").format.rowHeight = 36;
  sheet.freezePanes.freezeRows(4);
  sheet.freezePanes.freezeColumns(3);
  const widths = { A: 7, B: 11, C: 16, D: 42, E: 35, F: 16, G: 13, H: 16, I: 14, J: 10, K: 18, L: 15, M: 10, N: 18, O: 13, P: 18, Q: 12, R: 17, S: 14, T: 10, U: 12, V: 8, W: 8, X: 8, Y: 8, Z: 8, AA: 14, AB: 12, AC: 34, AD: 34, AE: 16, AF: 30, AG: 44, AH: 21, AI: 34, AJ: 28 };
  for (const [col, width] of Object.entries(widths)) sheet.getRange(`${col}1:${col}${Math.max(endRow, 5)}`).format.columnWidth = width;
  return endRow;
}

const allEnd = setupDataSheet(all, "COCO Living Space｜全部商品清单", `共 ${joined.length} 件；每行保留原始文字、标准化数字、证据限制和状态。工作表不包含1688。`, joined, "AllProductsTable", true);
const highEnd = setupDataSheet(high, "累计总销量 ≥ 500 的已核验商品", `门槛字段：详情页累计总销量下限；当前 ${highSales.length} 件。近月销量与累计总销量分别保留，不混算。`, highSales, "HighSalesTable", true);
const deepEnd = setupDataSheet(detail, "有详情页、购物车或评论证据的商品", `当前 ${deep.length} 件；未取得的字段保持空白，不填零。`, deep, "DeepAuditTable", false);

async function addProductImages(sheet, rows) {
  let count = 0;
  for (let i = 0; i < rows.length; i++) {
    const imagePath = rows[i].product.imagePath;
    if (!imagePath) continue;
    try {
      const abs = path.resolve(runDir, imagePath);
      const bytes = await fs.readFile(abs);
      const mime = bytes[0] === 0xff && bytes[1] === 0xd8 ? "image/jpeg" : bytes[0] === 0x89 ? "image/png" : "image/webp";
      sheet.images.add({ dataUrl: `data:${mime};base64,${bytes.toString("base64")}`, anchor: { from: { row: i + 4, col: 1 }, extent: { widthPx: 58, heightPx: 58 } } });
      count++;
    } catch {}
  }
  return count;
}

const allSheetImageAnchors = await addProductImages(all, joined);
const highSalesImageAnchors = await addProductImages(high, highSales);
const embeddedImages = allSheetImageAnchors + highSalesImageAnchors;

const previews = [
  [overview, "概览", "A1:H12"],
  [all, "全部商品", `A1:AJ${Math.min(allEnd, 14)}`],
  [high, "高销量商品", `A1:AJ${Math.min(highEnd, 14)}`],
  [detail, "深度核验", `A1:AJ${Math.min(deepEnd, 14)}`]
];
for (const [, name, range] of previews) {
  const blob = await wb.render({ sheetName: name, range, scale: 1, format: "png" });
  await fs.writeFile(path.join(runDir, `preview_${name}.png`), new Uint8Array(await blob.arrayBuffer()));
}

const imagePreviews = [
  ["全部商品", `A1:F${Math.min(allEnd, 14)}`],
  ["高销量商品", `A1:F${Math.min(highEnd, 14)}`]
];
for (const [name, range] of imagePreviews) {
  const blob = await wb.render({ sheetName: name, range, scale: 1.5, format: "png" });
  await fs.writeFile(path.join(runDir, `preview_${name}_图片检查.png`), new Uint8Array(await blob.arrayBuffer()));
}

const overviewInspect = await wb.inspect({ kind: "table", range: "概览!A1:H12", include: "values,formulas", tableMaxRows: 20, tableMaxCols: 10 });
const errors = await wb.inspect({ kind: "match", searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A|#NUM!|#NULL!", options: { useRegex: true, maxResults: 300 }, summary: "final formula error scan" });
const out = await SpreadsheetFile.exportXlsx(wb);
await out.save(outputPath);
const savedWb = await SpreadsheetFile.importXlsx(await FileBlob.load(outputPath));
const savedHighPreview = await savedWb.render({ sheetName: "高销量商品", range: `A1:F${Math.min(highEnd, 14)}`, scale: 1.5, format: "png" });
await fs.writeFile(path.join(runDir, "preview_高销量商品_导出复核.png"), new Uint8Array(await savedHighPreview.arrayBuffer()));
const savedErrors = await savedWb.inspect({ kind: "match", searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A|#NUM!|#NULL!", options: { useRegex: true, maxResults: 300 }, summary: "exported workbook formula error scan" });

const verification = {
  verifiedAt: new Date().toISOString(),
  validState: true,
  workbookPath: outputPath,
  catalogRows: catalog.products.length,
  uniqueItemIds: new Set(catalog.products.map((p) => p.itemId)).size,
  imageAnchors: embeddedImages,
  expectedImages: catalog.products.filter((p) => p.imagePath).length + highSales.filter((x) => x.product.imagePath).length,
  allSheetImageAnchors,
  allSheetExpectedImages: catalog.products.filter((p) => p.imagePath).length,
  highSalesImageAnchors,
  highSalesExpectedImages: highSales.filter((x) => x.product.imagePath).length,
  statusCounts,
  detailTotalSalesCount: detailCount,
  safeCartPriceCount: cartCount,
  reviewSummaryCount: reviewCount,
  completeReviewCount,
  highSalesThreshold: 500,
  highSalesSourceField: "detail totalSoldLower",
  highSalesRows: highSales.length,
  formulaErrorScan: errors.ndjson,
  exportedFormulaErrorScan: savedErrors.ndjson,
  exportedWorkbookReopened: true,
  overviewInspection: overviewInspect.ndjson,
  renderedSheets: previews.map(([, name]) => name),
  imagePreviewSheets: imagePreviews.map(([name]) => name)
};
await fs.writeFile(path.join(runDir, "verification.json"), JSON.stringify(verification, null, 2) + "\n");
console.log(JSON.stringify({ outputPath, embeddedImages, allSheetImageAnchors, highSalesImageAnchors, allRows: joined.length, highRows: highSales.length, deepRows: deep.length }, null, 2));
