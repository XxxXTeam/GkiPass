/**
 * 将 Next.js 静态导出 (out/) 复制到 plane/frontend/out/
 * 功能：构建后自动同步前端产物到 Go 嵌入目录
 */
const fs = require("fs");
const path = require("path");

const src = path.join(__dirname, "..", "out");
const dst = path.join(__dirname, "..", "..", "plane", "frontend", "out");

/* 1. 清空目标目录 */
fs.rmSync(dst, { recursive: true, force: true });
fs.mkdirSync(dst, { recursive: true });

/* 2. 递归复制 */
function copyDir(s, d) {
  for (const f of fs.readdirSync(s)) {
    const sp = path.join(s, f);
    const dp = path.join(d, f);
    if (fs.statSync(sp).isDirectory()) {
      fs.mkdirSync(dp, { recursive: true });
      copyDir(sp, dp);
    } else {
      fs.copyFileSync(sp, dp);
    }
  }
}
copyDir(src, dst);

/* 3. 写入 .gitignore，防止构建产物提交到仓库 */
fs.writeFileSync(
  path.join(dst, ".gitignore"),
  "# 前端构建产物不提交，由构建流程自动复制\n*\n!.gitignore\n!PLACEHOLDER\n"
);

console.log("✓ out/ -> plane/frontend/out/");
