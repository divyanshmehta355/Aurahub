import { NextResponse } from "next/server";
import { generateIdenticonSvg } from "@/lib/identicon";

export const dynamic = "force-dynamic";

export async function GET(request, { params }) {
  try {
    const { seed } = await params;
    const decodedSeed = decodeURIComponent(seed || "user");
    const svg = generateIdenticonSvg(decodedSeed);

    return new NextResponse(svg, {
      status: 200,
      headers: {
        "Content-Type": "image/svg+xml; charset=utf-8",
        "Cache-Control": "public, max-age=31536000, immutable",
      },
    });
  } catch (error) {
    return new NextResponse("Error generating avatar", { status: 500 });
  }
}
