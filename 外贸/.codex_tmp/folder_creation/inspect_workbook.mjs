import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";
import { fileURLToPath } from "node:url";

const workbookPath = fileURLToPath(
  new URL(
    "../../市场分析/东南亚市场/Shopee/店铺选品/COCO_Living_Space/COCO_Living_Space_Shopee全量审计_持续更新.xlsx",
    import.meta.url,
  ),
);

const input = await FileBlob.load(workbookPath);
const workbook = await SpreadsheetFile.importXlsx(input);

const sheet = workbook.worksheets.getItem("高销量商品");
const rows = sheet.getRange("A1:E86").values;

console.log(JSON.stringify(rows.map((values, index) => ({
  excelRow: index + 1,
  sequence: values[0],
  productId: values[2],
  chineseName: values[3],
  englishTitle: values[4],
})).filter((row) => row.excelRow >= 5 && !row.chineseName), null, 2));
