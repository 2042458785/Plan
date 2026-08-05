import fs from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";

const cocoRoot = fileURLToPath(
  new URL(
    "../../市场分析/东南亚市场/Shopee/店铺选品/COCO_Living_Space/",
    import.meta.url,
  ),
);
const workbookPath = `${cocoRoot}COCO_Living_Space_Shopee全量审计_持续更新.xlsx`;
const targetDir = `${cocoRoot}高销量商品调研`;

const translatedFallbacks = new Map([
  [8, "强力磁吸门后挂钩免打孔"],
  [11, "ins工业风304不锈钢方形托盘"],
  [22, "韩式不规则不锈钢腰果首饰托盘"],
  [47, "浴室防水防雾吸盘静音小挂钟"],
  [57, "简约不锈钢浴室沥水置物架"],
  [63, "强力磁吸门后承重挂钩免打孔"],
  [80, "中古风实木首饰香水收纳托盘"],
]);

function sanitizeName(value) {
  return String(value)
    .trim()
    .replace(/[\\/:*?"<>|]/g, "-")
    .replace(/\s+/g, " ");
}

const input = await FileBlob.load(workbookPath);
const workbook = await SpreadsheetFile.importXlsx(input);
const sheet = workbook.worksheets.getItem("高销量商品");
const values = sheet.getRange("A5:D86").values;

const products = values.map((row, offset) => {
  const excelRow = offset + 5;
  const productId = String(row[2] ?? "").trim();
  const spreadsheetName = String(row[3] ?? "").trim();
  const chineseName = spreadsheetName || translatedFallbacks.get(excelRow) || "";

  if (!/^\d+$/.test(productId)) {
    throw new Error(`Excel 第 ${excelRow} 行商品ID无效：${productId}`);
  }
  if (!chineseName) {
    throw new Error(`Excel 第 ${excelRow} 行缺少商品中文名称，且没有备用译名`);
  }

  return {
    excelRow,
    productId,
    chineseName,
    usedFallbackTranslation: !spreadsheetName,
    folderName: `${excelRow}_${productId}_${sanitizeName(chineseName)}`,
  };
});

if (products.length !== 82) {
  throw new Error(`预期 82 个商品，实际读取 ${products.length} 个`);
}

const uniqueFolders = new Set(products.map((item) => item.folderName));
if (uniqueFolders.size !== products.length) {
  throw new Error("目标文件夹名称存在重复，已停止创建");
}

await fs.mkdir(targetDir, { recursive: true });
const currentEntries = await fs.readdir(targetDir, { withFileTypes: true });
const currentDirs = new Set(
  currentEntries.filter((entry) => entry.isDirectory()).map((entry) => entry.name),
);

const report = {
  targetDir,
  total: products.length,
  created: [],
  renamed: [],
  unchanged: [],
  translatedFallbacks: [],
  conflicts: [],
};

for (const product of products) {
  const { excelRow, productId, folderName, usedFallbackTranslation } = product;
  const prefix = `${excelRow}_${productId}_`;

  if (usedFallbackTranslation) {
    report.translatedFallbacks.push(folderName);
  }

  if (currentDirs.has(folderName)) {
    report.unchanged.push(folderName);
    continue;
  }

  const prefixMatches = [...currentDirs].filter((name) => name.startsWith(prefix));
  if (prefixMatches.length === 1) {
    const oldName = prefixMatches[0];
    await fs.rename(`${targetDir}/${oldName}`, `${targetDir}/${folderName}`);
    currentDirs.delete(oldName);
    currentDirs.add(folderName);
    report.renamed.push({ from: oldName, to: folderName });
    continue;
  }

  if (prefixMatches.length > 1) {
    report.conflicts.push({ expected: folderName, existing: prefixMatches });
    continue;
  }

  await fs.mkdir(`${targetDir}/${folderName}`);
  currentDirs.add(folderName);
  report.created.push(folderName);
}

if (process.argv.includes("--summary")) {
  console.log(JSON.stringify({
    targetDir: report.targetDir,
    expected: report.total,
    created: report.created.length,
    renamed: report.renamed.length,
    unchanged: report.unchanged.length,
    translatedFallbacks: report.translatedFallbacks.length,
    conflicts: report.conflicts.length,
    actualDirectories: (await fs.readdir(targetDir, { withFileTypes: true }))
      .filter((entry) => entry.isDirectory()).length,
  }, null, 2));
} else {
  console.log(JSON.stringify(report, null, 2));
}

if (report.conflicts.length > 0) {
  process.exitCode = 2;
}
