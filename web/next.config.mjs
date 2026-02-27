/** @type {import('next').NextConfig} */
const nextConfig = {
  /* 静态导出模式：生成纯静态 HTML/JS/CSS，嵌入 Go 二进制 */
  output: "export",

  /* 尾部斜杠：确保 /dashboard/ 映射到 /dashboard/index.html */
  trailingSlash: true,

  /* 静态导出不支持 Next.js 内置图片优化 */
  images: { unoptimized: true },

  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  },
};

export default nextConfig;
