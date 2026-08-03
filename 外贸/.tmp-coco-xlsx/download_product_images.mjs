import fs from "node:fs/promises";
import path from "node:path";

const datasetPath = "/private/tmp/coco_living_space_dataset.json";
const outputDir = "/Users/ddy/GolandProjects/Plans/Business-Plan/外贸/outputs/coco-living-space-20260730/product_images";
const indexPath = "/Users/ddy/GolandProjects/Plans/Business-Plan/外贸/outputs/coco-living-space-20260730/product_image_index.json";
const dataset = JSON.parse(await fs.readFile(datasetPath, "utf8"));
await fs.mkdir(outputDir, { recursive: true });

const products = dataset.allProducts;
const concurrency = 8;
let cursor = 0;
let completed = 0;
const records = new Array(products.length);

async function downloadOne(product, index) {
  const fileName = `${product.itemId}.webp`;
  const filePath = path.join(outputDir, fileName);
  try {
    const stat = await fs.stat(filePath).catch(() => null);
    if (stat?.size > 1000) {
      records[index] = { itemId: product.itemId, status: "existing", filePath, bytes: stat.size, sourceUrl: product.image };
      return;
    }
    if (!product.image) {
      records[index] = { itemId: product.itemId, status: "missing_url", filePath: "", bytes: 0, sourceUrl: "" };
      return;
    }
    let lastError = "";
    for (let attempt = 1; attempt <= 3; attempt++) {
      try {
        const response = await fetch(product.image, {
          headers: {
            "User-Agent": "Mozilla/5.0",
            Referer: "https://shopee.com.my/",
          },
          redirect: "follow",
        });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const bytes = new Uint8Array(await response.arrayBuffer());
        if (bytes.length < 1000) throw new Error(`small response ${bytes.length}`);
        await fs.writeFile(filePath, bytes);
        records[index] = {
          itemId: product.itemId,
          status: "downloaded",
          filePath,
          bytes: bytes.length,
          sourceUrl: product.image,
          contentType: response.headers.get("content-type") || "",
        };
        return;
      } catch (error) {
        lastError = String(error?.message || error);
      }
    }
    records[index] = { itemId: product.itemId, status: "failed", filePath: "", bytes: 0, sourceUrl: product.image, error: lastError };
  } finally {
    completed++;
    if (completed % 50 === 0 || completed === products.length) {
      console.log(`progress ${completed}/${products.length}`);
    }
  }
}

async function worker() {
  while (true) {
    const index = cursor++;
    if (index >= products.length) return;
    await downloadOne(products[index], index);
  }
}

await Promise.all(Array.from({ length: concurrency }, () => worker()));
const summary = {
  total: records.length,
  downloaded: records.filter((r) => r?.status === "downloaded").length,
  existing: records.filter((r) => r?.status === "existing").length,
  failed: records.filter((r) => r?.status === "failed").length,
  missingUrl: records.filter((r) => r?.status === "missing_url").length,
};
await fs.writeFile(indexPath, JSON.stringify({ summary, records }, null, 2));
console.log(JSON.stringify({ summary, outputDir, indexPath }));
