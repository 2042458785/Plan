import fs from "node:fs/promises";
import { SpreadsheetFile, Workbook } from "@oai/artifact-tool";
import sharp from "sharp";

const datasetPath = "/private/tmp/coco_living_space_dataset.json";
const outputDir = "/Users/ddy/GolandProjects/Plans/Business-Plan/外贸/outputs/coco-living-space-20260730";
const outputPath =
  process.env.COCO_OUTPUT_PATH ||
  `${outputDir}/COCO_Living_Space_高销量商品与1688搜货表.xlsx`;
const data = JSON.parse(await fs.readFile(datasetPath, "utf8"));
const detailBundle = JSON.parse(await fs.readFile(`${outputDir}/shopee_detail_results.json`, "utf8"));
const detailResults = detailBundle.results;
let progressResults = {};
try {
  const progressBundle = JSON.parse(
    await fs.readFile(`${outputDir}/shopee_progress_results.json`, "utf8"),
  );
  progressResults = progressBundle.results || {};
} catch (error) {
  if (error?.code !== "ENOENT") throw error;
}
const imageDir = `${outputDir}/product_images`;

const workbook = Workbook.create();
const summary = workbook.worksheets.add("概览");
const high = workbook.worksheets.add("高销量商品");
const verified = workbook.worksheets.add("高销量深度核验");
const all = workbook.worksheets.add("全部商品");

for (const sheet of [summary, high, verified, all]) {
  sheet.showGridLines = false;
}

const navy = "#203248";
const bronze = "#B68A55";
const cream = "#F7F2EA";
const lightBlue = "#EAF0F6";
const lightGreen = "#E8F3EC";
const lightAmber = "#FFF2D8";
const lightRed = "#FCE8E6";
const textColor = "#243242";
const muted = "#657384";
const border = "#D8DEE6";

// High-sales shortlist.
high.mergeCells("A1:R1");
high.getRange("A1").values = [["COCO Living Space｜累计销量 ≥500 商品与 1688 搜货表"]];
high.getRange("A1:R1").format = {
  fill: navy,
  font: { bold: true, color: "#FFFFFF", size: 16 },
  horizontalAlignment: "left",
  verticalAlignment: "center",
};
high.getRange("A1:R1").format.rowHeight = 32;

high.mergeCells("A2:R2");
high.getRange("A2").values = [[
  "销量为 Shopee 店铺装修商品卡片公开显示的累计销量；“1k+”等为平台近似值。1688 成本区保留待填，中文关键词可直接复制搜索。",
]];
high.getRange("A2:R2").format = {
  fill: cream,
  font: { color: muted, italic: true, size: 10 },
  wrapText: true,
  verticalAlignment: "center",
};
high.getRange("A2:R2").format.rowHeight = 34;

const highHeaders = [
  "序号",
  "销量分组",
  "Shopee商品ID",
  "商品英文标题",
  "1688中文搜货词",
  "Shopee累计销量显示",
  "销量下限",
  "Shopee售价(MYR)",
  "1688匹配状态",
  "1688商品标题",
  "1688成本低(CNY)",
  "1688成本高(CNY)",
  "起订量",
  "匹配置信度",
  "Shopee链接",
  "1688链接",
  "商品图片链接",
  "备注",
];
high.getRange("A4:R4").values = [highHeaders];
high.getRange("A4:R4").format = {
  fill: bronze,
  font: { bold: true, color: "#FFFFFF", size: 10 },
  horizontalAlignment: "center",
  verticalAlignment: "center",
  wrapText: true,
  borders: { bottom: { style: "medium", color: navy } },
};
high.getRange("A4:R4").format.rowHeight = 38;

const highRows = data.highSales.map((p) => [
  p.rank,
  p.salesBand,
  p.itemId,
  p.name,
  p.cnSearch,
  p.soldRaw,
  p.soldLower,
  p.priceMin,
  "待匹配（需打开1688页面）",
  "",
  null,
  null,
  null,
  "待核验",
  p.url,
  "",
  p.image,
  "",
]);
const highEnd = 4 + highRows.length;
high.getRange(`A5:R${highEnd}`).values = highRows;
high.getRange(`A5:R${highEnd}`).format = {
  font: { color: textColor, size: 10 },
  verticalAlignment: "center",
  borders: { insideHorizontal: { style: "thin", color: border } },
};
high.getRange(`D5:E${highEnd}`).format.wrapText = true;
high.getRange(`I5:J${highEnd}`).format.wrapText = true;
high.getRange(`O5:R${highEnd}`).format.wrapText = true;
high.getRange(`A5:C${highEnd}`).format.horizontalAlignment = "center";
high.getRange(`F5:I${highEnd}`).format.horizontalAlignment = "center";
high.getRange(`K5:N${highEnd}`).format.horizontalAlignment = "center";
high.getRange(`G5:G${highEnd}`).format.numberFormat = "#,##0";
high.getRange(`H5:H${highEnd}`).format.numberFormat = "\"RM\" #,##0.00";
high.getRange(`K5:L${highEnd}`).format.numberFormat = "\"¥\" #,##0.00";
high.getRange(`A5:R${highEnd}`).format.rowHeight = 50;
high.getRange(`G5:G${highEnd}`).conditionalFormats.add("cellIs", {
  operator: "greaterThanOrEqual",
  formula: 1000,
  format: { fill: lightGreen, font: { bold: true, color: "#27633B" } },
});
high.getRange(`G5:G${highEnd}`).conditionalFormats.add("cellIs", {
  operator: "between",
  formula: [500, 999],
  format: { fill: lightAmber, font: { color: "#80570E" } },
});
high.getRange(`I5:I${highEnd}`).conditionalFormats.add("containsText", {
  text: "待匹配",
  format: { fill: lightRed, font: { color: "#A33B32" } },
});
high.getRange(`I5:I${highEnd}`).dataValidation = {
  rule: {
    type: "list",
    values: ["待匹配（需打开1688页面）", "已匹配", "需复核", "未找到同款"],
  },
};
high.getRange(`N5:N${highEnd}`).dataValidation = {
  rule: { type: "list", values: ["高", "中", "低", "待核验"] },
};
high.tables.add(`A4:R${highEnd}`, true, "HighSalesProducts");
high.freezePanes.freezeRows(4);
high.freezePanes.freezeColumns(3);

const highWidths = {
  A: 7, B: 17, C: 16, D: 48, E: 35, F: 19, G: 12, H: 17, I: 24,
  J: 40, K: 17, L: 17, M: 11, N: 13, O: 34, P: 34, Q: 34, R: 28,
};
for (const [col, width] of Object.entries(highWidths)) {
  high.getRange(`${col}:${col}`).format.columnWidth = width;
}

// Detail-verified shortlist with embedded product images.
const parseAbbreviatedNumber = (raw) => {
  const match = String(raw || "").match(/([\d.]+)\s*([kKmM]?)/);
  if (!match) return null;
  const factor = match[2].toLowerCase() === "k" ? 1000 : match[2].toLowerCase() === "m" ? 1000000 : 1;
  return Math.round(Number(match[1]) * factor);
};
const parseMoneyRange = (raw) => {
  const values = [...String(raw || "").matchAll(/RM\s*([\d,.]+)/gi)].map((m) => Number(m[1].replace(/,/g, "")));
  return { low: values[0] ?? null, high: values[1] ?? values[0] ?? null };
};

verified.mergeCells("A1:Z1");
verified.getRange("A1").values = [["COCO Living Space｜高销量商品详情页深度核验"]];
verified.getRange("A1:Z1").format = {
  fill: navy,
  font: { bold: true, color: "#FFFFFF", size: 16 },
  verticalAlignment: "center",
};
verified.getRange("A1:Z1").format.rowHeight = 32;
verified.mergeCells("A2:Z2");
verified.getRange("A2").values = [[
  "月销量来自店铺 All Products 卡片的 Sold/Month；总销量、详情现价与划线原价来自商品详情页。10件金额为“最低详情现价×10”的理论值，不是购物车或结账价。",
]];
verified.getRange("A2:Z2").format = {
  fill: cream,
  font: { color: muted, italic: true, size: 10 },
  wrapText: true,
  verticalAlignment: "center",
};
verified.getRange("A2:Z2").format.rowHeight = 34;
const verifiedHeaders = [
  "序号",
  "Shopee商品ID",
  "商品主图",
  "商品英文标题",
  "月销量显示",
  "月销量",
  "总销量显示",
  "总销量下限",
  "详情现价文本",
  "现价最低(MYR)",
  "现价最高(MYR)",
  "划线原价(MYR)",
  "10件理论总额(MYR)",
  "10件理论均价(MYR)",
  "价格口径说明",
  "本地主图路径",
  "Shopee链接",
  "详情核验状态",
  "1688匹配状态",
  "1688商品标题",
  "1688成本低(CNY)",
  "1688成本高(CNY)",
  "起订量",
  "匹配置信度",
  "1688链接",
  "备注",
];
verified.getRange("A4:Z4").values = [verifiedHeaders];
verified.getRange("A4:Z4").format = {
  fill: bronze,
  font: { bold: true, color: "#FFFFFF", size: 9 },
  horizontalAlignment: "center",
  verticalAlignment: "center",
  wrapText: true,
  borders: { bottom: { style: "medium", color: navy } },
};
verified.getRange("A4:Z4").format.rowHeight = 42;

const verifiedRows = data.highSales.map((product) => {
  const detail = detailResults[product.itemId] || {};
  const range = parseMoneyRange(detail.priceRaw);
  const original = parseMoneyRange(detail.originalPriceRaw).low;
  const hasRange = range.low != null && range.high != null && range.low !== range.high;
  return [
    product.rank,
    product.itemId,
    "",
    product.name,
    detail.monthlyRaw || "",
    detail.monthlySold ?? null,
    detail.totalSoldRaw || "",
    parseAbbreviatedNumber(detail.totalSoldRaw),
    detail.priceRaw || "",
    range.low,
    range.high,
    original,
    null,
    null,
    hasRange
      ? "存在变体价范围；10件理论值按最低价计算，未选择具体规格。"
      : "详情页当前商品价；未扣店铺券、平台券、运费或结账优惠。",
    `${imageDir}/${product.itemId}.webp`,
    product.url,
    detail.totalSoldRaw && detail.priceRaw ? "已核验详情页" : "待重试",
    "待图片搜索（需打开1688）",
    "",
    null,
    null,
    null,
    "待核验",
    "",
    "",
  ];
});
const verifiedEnd = 4 + verifiedRows.length;
verified.getRange(`A5:Z${verifiedEnd}`).values = verifiedRows;
verified.getRange("M5").formulas = [["=IF(J5=\"\",\"\",J5*10)"]];
verified.getRange(`M5:M${verifiedEnd}`).fillDown();
verified.getRange("N5").formulas = [["=IF(M5=\"\",\"\",M5/10)"]];
verified.getRange(`N5:N${verifiedEnd}`).fillDown();
verified.getRange(`A5:Z${verifiedEnd}`).format = {
  font: { color: textColor, size: 9 },
  verticalAlignment: "center",
  borders: { insideHorizontal: { style: "thin", color: border } },
};
verified.getRange(`A5:B${verifiedEnd}`).format.horizontalAlignment = "center";
verified.getRange(`E5:N${verifiedEnd}`).format.horizontalAlignment = "center";
verified.getRange(`R5:X${verifiedEnd}`).format.horizontalAlignment = "center";
verified.getRange(`D5:D${verifiedEnd}`).format.wrapText = true;
verified.getRange(`O5:Q${verifiedEnd}`).format.wrapText = true;
verified.getRange(`S5:Z${verifiedEnd}`).format.wrapText = true;
verified.getRange(`F5:F${verifiedEnd}`).format.numberFormat = "#,##0";
verified.getRange(`H5:H${verifiedEnd}`).format.numberFormat = "#,##0";
verified.getRange(`J5:N${verifiedEnd}`).format.numberFormat = "\"RM\" #,##0.00";
verified.getRange(`U5:V${verifiedEnd}`).format.numberFormat = "\"¥\" #,##0.00";
verified.getRange(`A5:Z${verifiedEnd}`).format.rowHeight = 84;
verified.getRange(`R5:R${verifiedEnd}`).conditionalFormats.add("containsText", {
  text: "已核验",
  format: { fill: lightGreen, font: { color: "#27633B", bold: true } },
});
verified.getRange(`S5:S${verifiedEnd}`).conditionalFormats.add("containsText", {
  text: "待图片搜索",
  format: { fill: lightRed, font: { color: "#A33B32" } },
});
verified.getRange(`S5:S${verifiedEnd}`).dataValidation = {
  rule: { type: "list", values: ["待图片搜索（需打开1688）", "已匹配", "需复核", "未找到同款"] },
};
verified.getRange(`X5:X${verifiedEnd}`).dataValidation = {
  rule: { type: "list", values: ["高", "中", "低", "待核验"] },
};
verified.tables.add(`A4:Z${verifiedEnd}`, true, "VerifiedHighSalesProducts");
verified.freezePanes.freezeRows(4);
verified.freezePanes.freezeColumns(4);

const verifiedWidths = {
  A: 7, B: 16, C: 15, D: 44, E: 18, F: 11, G: 16, H: 13, I: 20,
  J: 16, K: 16, L: 16, M: 20, N: 20, O: 38, P: 38, Q: 38, R: 17,
  S: 25, T: 40, U: 17, V: 17, W: 11, X: 13, Y: 38, Z: 30,
};
for (const [col, width] of Object.entries(verifiedWidths)) {
  verified.getRange(`${col}:${col}`).format.columnWidth = width;
}
for (let index = 0; index < data.highSales.length; index++) {
  const itemId = data.highSales[index].itemId;
  const imagePath = `${imageDir}/${itemId}.webp`;
  const bytes = await fs.readFile(imagePath);
  const pngBytes = await sharp(bytes)
    .resize(160, 160, { fit: "contain", background: { r: 255, g: 255, b: 255, alpha: 1 } })
    .png()
    .toBuffer();
  verified.images.add({
    dataUrl: `data:image/png;base64,${pngBytes.toString("base64")}`,
    anchor: {
      from: { row: index + 4, col: 2, rowOffsetPx: 3, colOffsetPx: 3 },
      extent: { widthPx: 76, heightPx: 76 },
    },
  });
}

// Full product catalogue and item-by-item verification ledger.
all.mergeCells("A1:Z1");
all.getRange("A1").values = [["COCO Living Space｜全部商品清单"]];
all.getRange("A1:Z1").format = {
  fill: navy,
  font: { bold: true, color: "#FFFFFF", size: 16 },
  verticalAlignment: "center",
};
all.getRange("A1:L1").format.rowHeight = 32;
all.mergeCells("A2:Z2");
all.getRange("A2").values = [[
  "共抓取 30 页、885 件商品；每行已嵌入对应主图。近月销量来自店铺列表页；总销量、10件实际价格和评论摘要必须逐件进入详情页核验，未核验字段保持空白。",
]];
all.getRange("A2:Z2").format = {
  fill: cream,
  font: { color: muted, italic: true, size: 10 },
  wrapText: true,
  verticalAlignment: "center",
};
all.getRange("A2:Z2").format.rowHeight = 32;
const allHeaders = [
  "序号",
  "商品主图",
  "店铺页",
  "页内序号",
  "Shopee商品ID",
  "商品英文标题",
  "Shopee售价(MYR)",
  "近月销量",
  "月销量显示",
  "评分",
  "总销量显示",
  "总销量下限",
  "详情现价文本",
  "10件实际总额(MYR)",
  "10件实际均价(MYR)",
  "评论样本数",
  "好评摘要",
  "差评摘要",
  "总销量核验",
  "10件价格核验",
  "评论核验",
  "证据置信度",
  "核验日期",
  "核验备注",
  "Shopee链接",
  "本地主图文件",
];
all.getRange("A4:Z4").values = [allHeaders];
all.getRange("A4:Z4").format = {
  fill: bronze,
  font: { bold: true, color: "#FFFFFF", size: 10 },
  horizontalAlignment: "center",
  verticalAlignment: "center",
  wrapText: true,
  borders: { bottom: { style: "medium", color: navy } },
};
all.getRange("A4:Z4").format.rowHeight = 42;
const allRows = data.allProducts.map((p, index) => {
  const detail = detailResults[p.itemId] || {};
  const progress = progressResults[p.itemId] || {};
  const totalSoldRaw = progress.totalSoldRaw || detail.totalSoldRaw || "";
  const hasTotalSoldEvidence = Boolean(totalSoldRaw);
  return [
    index + 1,
    "",
    p.page,
    p.pageIndex,
    p.itemId,
    p.name,
    p.priceMin,
    p.monthSold,
    p.monthRaw,
    p.rating,
    totalSoldRaw,
    progress.totalSoldLower ?? parseAbbreviatedNumber(totalSoldRaw),
    progress.detailPriceRaw || detail.priceRaw || "",
    progress.cartLineTotal ?? null,
    null,
    progress.visibleReviewSampleCount ?? null,
    progress.positiveSummary || "",
    progress.negativeSummary || "",
    progress.totalSalesStatus || (hasTotalSoldEvidence ? "已核验详情页" : "待核验"),
    progress.quantityPriceStatus || "待实测",
    progress.reviewStatus || "待核验",
    progress.confidence ||
      (hasTotalSoldEvidence
        ? "高：显示文字；中高：销量下限"
        : "未知：未访问详情页"),
    progress.verifiedDate || (hasTotalSoldEvidence ? "2026-07-30" : ""),
    progress.evidenceNote ||
      (hasTotalSoldEvidence
        ? "总销量与详情现价来自详情页；10件实际价格和评论摘要未完成。"
        : ""),
    p.url,
    `${p.itemId}.webp`,
  ];
});
const detailVerifiedCount = allRows.filter((row) => row[18] === "已核验详情页").length;
const allEnd = 4 + allRows.length;
all.getRange(`A5:Z${allEnd}`).values = allRows;
all.getRange("O5").formulas = [["=IF(N5=\"\",\"\",N5/10)"]];
all.getRange(`O5:O${allEnd}`).fillDown();
all.getRange(`A5:Z${allEnd}`).format = {
  font: { color: textColor, size: 9 },
  verticalAlignment: "center",
  borders: { insideHorizontal: { style: "thin", color: "#E8EBEF" } },
};
all.getRange(`F5:F${allEnd}`).format.wrapText = true;
all.getRange(`K5:M${allEnd}`).format.wrapText = true;
all.getRange(`Q5:R${allEnd}`).format.wrapText = true;
all.getRange(`V5:Y${allEnd}`).format.wrapText = true;
all.getRange(`A5:E${allEnd}`).format.horizontalAlignment = "center";
all.getRange(`G5:P${allEnd}`).format.horizontalAlignment = "center";
all.getRange(`S5:W${allEnd}`).format.horizontalAlignment = "center";
all.getRange(`G5:G${allEnd}`).format.numberFormat = "\"RM\" #,##0.00";
all.getRange(`H5:H${allEnd}`).format.numberFormat = "#,##0";
all.getRange(`J5:J${allEnd}`).format.numberFormat = "0.0";
all.getRange(`L5:L${allEnd}`).format.numberFormat = "#,##0";
all.getRange(`N5:O${allEnd}`).format.numberFormat = "\"RM\" #,##0.00";
all.getRange(`P5:P${allEnd}`).format.numberFormat = "#,##0";
all.getRange(`W5:W${allEnd}`).format.numberFormat = "yyyy-mm-dd";
all.getRange(`A5:Z${allEnd}`).format.rowHeight = 76;
all.getRange(`H5:H${allEnd}`).conditionalFormats.add("dataBar", {
  color: "#6A9A7C",
  gradient: true,
});
all.getRange(`S5:S${allEnd}`).conditionalFormats.add("containsText", {
  text: "已核验",
  format: { fill: lightGreen, font: { color: "#27633B", bold: true } },
});
all.getRange(`T5:U${allEnd}`).conditionalFormats.add("containsText", {
  text: "待",
  format: { fill: lightAmber, font: { color: "#80570E" } },
});
all.getRange(`S5:U${allEnd}`).dataValidation = {
  rule: {
    type: "list",
    values: ["待核验", "待实测", "部分核验", "已核验详情页", "已核验", "无法访问", "需复核"],
  },
};
all.tables.add(`A4:Z${allEnd}`, true, "AllProducts");
all.freezePanes.freezeRows(4);
all.freezePanes.freezeColumns(4);

const allWidths = {
  A: 8, B: 15, C: 9, D: 11, E: 17, F: 52, G: 18, H: 13, I: 18, J: 9,
  K: 17, L: 14, M: 20, N: 20, O: 20, P: 14, Q: 38, R: 38, S: 18, T: 17,
  U: 15, V: 30, W: 15, X: 42, Y: 32, Z: 22,
};
for (const [col, width] of Object.entries(allWidths)) {
  all.getRange(`${col}:${col}`).format.columnWidth = width;
}
let allEmbeddedImageCount = 0;
for (let index = 0; index < data.allProducts.length; index++) {
  const itemId = data.allProducts[index].itemId;
  const imagePath = `${imageDir}/${itemId}.webp`;
  const bytes = await fs.readFile(imagePath);
  const jpegBytes = await sharp(bytes)
    .resize(120, 120, {
      fit: "contain",
      background: { r: 255, g: 255, b: 255, alpha: 1 },
      withoutEnlargement: true,
    })
    .jpeg({ quality: 74, mozjpeg: true })
    .toBuffer();
  all.images.add({
    dataUrl: `data:image/jpeg;base64,${jpegBytes.toString("base64")}`,
    anchor: {
      from: { row: index + 4, col: 1, rowOffsetPx: 3, colOffsetPx: 3 },
      extent: { widthPx: 68, heightPx: 68 },
    },
  });
  allEmbeddedImageCount += 1;
}

// Overview and QA guide.
summary.mergeCells("A1:H1");
summary.getRange("A1").values = [["COCO Living Space 选品数据概览"]];
summary.getRange("A1:H1").format = {
  fill: navy,
  font: { bold: true, color: "#FFFFFF", size: 18 },
  verticalAlignment: "center",
};
summary.getRange("A1:H1").format.rowHeight = 36;
summary.mergeCells("A2:H2");
summary.getRange("A2").values = [[
  `采集日期：2026-07-30｜店铺：${data.shop.name}｜Shopee Malaysia`,
]];
summary.getRange("A2:H2").format = {
  fill: lightBlue,
  font: { color: muted, size: 10 },
  verticalAlignment: "center",
};

summary.getRange("A4:A9").values = [
  ["全部商品"],
  ["总销量已核验"],
  ["总销量待核验"],
  ["10件实价已核验"],
  ["评论已有摘要"],
  ["其中完整核验"],
];
summary.getRange("B4").formulas = [[`=COUNTA('全部商品'!$A$5:$A$${allEnd})`]];
summary.getRange("B5").formulas = [[`=COUNTIF('全部商品'!$S$5:$S$${allEnd},"已核验详情页")`]];
summary.getRange("B6").formulas = [["=B4-B5"]];
summary.getRange("B7").formulas = [[`=COUNT('全部商品'!$N$5:$N$${allEnd})`]];
summary.getRange("B8").formulas = [[`=COUNTIF('全部商品'!$U$5:$U$${allEnd},"已核验")+COUNTIF('全部商品'!$U$5:$U$${allEnd},"部分核验")`]];
summary.getRange("B9").formulas = [[`=COUNTIF('全部商品'!$U$5:$U$${allEnd},"已核验")`]];
summary.getRange("A4:A9").format = {
  fill: cream,
  font: { bold: true, color: textColor },
  verticalAlignment: "center",
};
summary.getRange("B4:B9").format = {
  fill: "#FFFFFF",
  font: { bold: true, color: navy, size: 16 },
  horizontalAlignment: "center",
  verticalAlignment: "center",
  numberFormat: "#,##0",
};
summary.getRange("A4:B9").format.borders = { preset: "outside", style: "thin", color: border };
summary.getRange("A4:B9").format.rowHeight = 30;

summary.mergeCells("D4:H4");
summary.getRange("D4").values = [["数据口径与当前限制"]];
summary.getRange("D4:H4").format = {
  fill: bronze,
  font: { bold: true, color: "#FFFFFF" },
  verticalAlignment: "center",
};
summary.mergeCells("D5:H9");
summary.getRange("D5").values = [[
  "1. 全部商品清单共 885 件，近月销量、列表起售价与评分已录入。\n" +
  `2. 其中 ${detailVerifiedCount} 件已有详情页总销量和详情现价证据；k+ 为平台近似显示，数值列仅记下限。\n` +
  `3. 其余 ${data.allProducts.length - detailVerifiedCount} 件尚未核验总销量，不能用近月销量替代。\n` +
  "4. “10件理论均价”不等于选择数量10后的真实结算前单价；实际字段在未实测前保持空白。\n" +
  "5. 评论好评/差评摘要必须基于详情页可见评论样本，当前未完成时保持空白。",
]];
summary.getRange("D5:H9").format = {
  fill: "#FFFFFF",
  font: { color: textColor, size: 10 },
  wrapText: true,
  verticalAlignment: "top",
  borders: { preset: "outside", style: "thin", color: border },
};

summary.mergeCells("A11:H11");
summary.getRange("A11").values = [["使用建议"]];
summary.getRange("A11:H11").format = {
  fill: navy,
  font: { bold: true, color: "#FFFFFF" },
  verticalAlignment: "center",
};
summary.mergeCells("A12:H15");
summary.getRange("A12").values = [[
  "逐件核验时每次只打开一个商品详情页，记录平台直接显示的总销量、详情价格、数量10的实际总额/均价和评论样本。无法访问、触发验证码或变体未选择时明确标记，不猜测、不用理论价格冒充实测价格。",
]];
summary.getRange("A12:H15").format = {
  fill: lightGreen,
  font: { color: textColor, size: 11 },
  wrapText: true,
  verticalAlignment: "top",
  borders: { preset: "outside", style: "thin", color: "#B7CEBF" },
};
summary.getRange("A:A").format.columnWidth = 22;
summary.getRange("B:B").format.columnWidth = 14;
for (const col of ["C", "D", "E", "F", "G", "H"]) {
  summary.getRange(`${col}:${col}`).format.columnWidth = 18;
}
summary.freezePanes.freezeRows(2);

await fs.mkdir(outputDir, { recursive: true });
const summaryPreview = await workbook.render({
  sheetName: "概览",
  range: "A1:H15",
  scale: 1.5,
  format: "png",
});
await fs.writeFile(`${outputDir}/preview_summary.png`, new Uint8Array(await summaryPreview.arrayBuffer()));
const highPreview = await workbook.render({
  sheetName: "高销量商品",
  range: "A1:R12",
  scale: 1,
  format: "png",
});
await fs.writeFile(`${outputDir}/preview_high_sales.png`, new Uint8Array(await highPreview.arrayBuffer()));
const verifiedPreview = await workbook.render({
  sheetName: "高销量深度核验",
  range: "A1:O10",
  scale: 1,
  format: "png",
});
await fs.writeFile(`${outputDir}/preview_verified_high_sales.png`, new Uint8Array(await verifiedPreview.arrayBuffer()));
const allPreview = await workbook.render({
  sheetName: "全部商品",
  range: "A1:V14",
  scale: 1,
  format: "png",
});
await fs.writeFile(`${outputDir}/preview_all_products.png`, new Uint8Array(await allPreview.arrayBuffer()));

const output = await SpreadsheetFile.exportXlsx(workbook);
await output.save(outputPath);

const checks = {
  embeddedImages: {
    allProducts: allEmbeddedImageCount,
    verifiedHighSales: data.highSales.length,
  },
  summary: (await workbook.inspect({
    kind: "table",
    range: "概览!A1:H15",
    include: "values,formulas",
    tableMaxRows: 20,
    tableMaxCols: 10,
  })).ndjson,
  high: (await workbook.inspect({
    kind: "table",
    range: `高销量商品!A1:N${highEnd}`,
    include: "values,formulas",
    tableMaxRows: 35,
    tableMaxCols: 14,
  })).ndjson,
  verified: (await workbook.inspect({
    kind: "table",
    range: `高销量深度核验!A1:Z${verifiedEnd}`,
    include: "values,formulas",
    tableMaxRows: 35,
    tableMaxCols: 26,
  })).ndjson,
  tail: (await workbook.inspect({
    kind: "table",
    range: `全部商品!A${allEnd - 2}:Z${allEnd}`,
    include: "values,formulas",
    tableMaxRows: 6,
    tableMaxCols: 26,
  })).ndjson,
  errors: (await workbook.inspect({
    kind: "match",
    searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A",
    options: { useRegex: true, maxResults: 100 },
    summary: "final formula error scan",
  })).ndjson,
};
await fs.writeFile(`${outputDir}/verification.json`, JSON.stringify(checks, null, 2));
console.log(JSON.stringify({
  outputPath,
  highEnd,
  allEnd,
  counts: {
    all: data.allProducts.length,
    high: data.highSales.length,
    embeddedAllProductImages: allEmbeddedImageCount,
  },
}));
