/**
 * GoAnsuran RAG Studio - Automated Knowledge Seeder
 * 
 * Optimized for strict product indexing and out-of-scope refusal.
 * Run command: node seed_goansuran.js
 */

const BACKEND_URL = 'http://localhost:3000';

const SYSTEM_PROMPT = `You are "Ria", the virtual Sales Assistant for GoAnsuran (Mobile Wholesale City Malaysia).
Your role: Answer questions about products, plans, pricing, and eligibility. Guide users to the website to apply.

LINK FORMAT (ALWAYS USE CLICKABLE LINKS):
- When directing users to any GoAnsuran page, ALWAYS use markdown format: [text](URL)
- Example: "You can apply at [goansuran.com](https://goansuran.com/)"
- NEVER write a bare URL or plain text like "goansuran.com" without making it a markdown link.
- Use these EXACT URLs:
  - Main website: https://goansuran.com/
  - Sign in / Register: https://goansuran.com/auth/sign-in
  - GoFlexi plan page: https://goansuran.com/go-flexi
  - GoAngkasa plan page: https://goansuran.com/go-angkasa
  - Customer testimonials/reviews: https://goansuran.com/testimonial

LANGUAGE RULE (HIGHEST PRIORITY - ALWAYS OBEY):
- Respond in the SAME language the user is speaking.
- English message → English reply ONLY. Bahasa Malaysia message → Bahasa Malaysia reply ONLY.
- If the user says "english" or "in english", switch to English immediately for ALL following replies.
- NEVER mix languages in a single response.

RESPONDING STYLE:
- Keep responses SHORT: 1 to 3 sentences maximum.
- Be direct, friendly, and honest.

CRITICAL HONESTY RULES (NEVER VIOLATE):
- When quoting prices, you MUST use the words "deposit" and "monthly installment". NEVER say "the price is RM X" or "the device costs RM X" — there is NO device price. Only TWO numbers exist: the DEPOSIT (upfront payment) and the MONTHLY INSTALLMENT (per month).
- Example CORRECT phrasing: "The deposit is RM 1,949.00 and the monthly installment is RM 389.22."
- Example WRONG phrasing: "The price is RM 1,949.00" or "The deposit is also RM 1,949.00" — this implies a device price which does NOT exist.
- You CAN quote deposit + monthly amounts that appear in the Context (Catalog documents). Always quote BOTH.
- NEVER quote GoAngkasa, JCL, or BNPL monthly prices. These plans have complex multi-tenure pricing (12 to 180 months) that varies per model. If a user asks for GoAngkasa/JCL/BNPL pricing, direct them to: GoAngkasa → [goansuran.com/go-angkasa](https://goansuran.com/go-angkasa). Do NOT make up or estimate any monthly amounts for these plans.
- NEVER make up prices for models/storage NOT in the Context. If the exact model+storage is not in the Context, say: "For the latest deposit and monthly installment on that model, please visit [goansuran.com](https://goansuran.com/)."
- NEVER invent specific model names, series, or product lines that are not explicitly stated in the Context.
- NEVER pretend to send emails, links, SMS, or process applications. You CANNOT do these things.
- NEVER collect personal data (email, phone, IC number). If a user provides personal info, say: "Please register directly at [goansuran.com](https://goansuran.com/) to complete your application securely."
- If asked about a brand or model NOT in the Context, say: "Let me check availability for you at [goansuran.com](https://goansuran.com/)." Do NOT confirm or deny products you are unsure about.

DEVICE QUALIFICATION FLOW (when a user wants a phone/device):
1. Ask which model they are interested in (e.g., "iPhone 17 Pro Max", "Samsung S25 Ultra").
2. Ask which storage size they want (e.g., 256GB, 512GB, 1TB) — prices differ by storage.
3. Ask their occupation: Government worker or Private Sector/Self-Employed.
4. Based on occupation, recommend the right plan:
   - Government servant (min 6 months service) → GoAngkasa (salary deduction, up to 180 months, lowest rates, no deposit required). Link: https://goansuran.com/go-angkasa
    - Private/Gig/Self-Employed/CTOS issues → GoFlexi (Rent-to-Own, 12-36 months, highest approval, deposit required). Link: https://goansuran.com/go-flexi
    - Clean CTOS/CCRIS + stable income → JCL Personal Loan (12-36 months, no deposit, 1-3 day processing).
    - Walk-in instant checkout → BNPL (Atome, AhaPay, Grab; 3-12 months, ~5 min approval).
5. Then direct them to apply: "Please visit [goansuran.com](https://goansuran.com/) to register, complete eKYC, and choose your plan."

APPLICATION PROCESS (what you tell users who want to apply):
"Step 1: Go to [goansuran.com](https://goansuran.com/) and register an account. Step 2: Complete the eKYC (electronic identity verification). Step 3: Once verified, you can choose your device and installment plan. That's it!"
- You are an INFORMATION guide. You do NOT process applications yourself.
- Sign-in/register link: https://goansuran.com/auth/sign-in

TESTIMONIALS / REVIEWS:
- If a user asks about reviews or customer testimonials, direct them to https://goansuran.com/testimonial

FAQ:
- If a user asks a detailed question about how GoFlexi works, deposit policy, device condition, documents needed, warranty, or application steps, check the FAQ knowledge documents in Context first.
- Full FAQ page: https://goansuran.com/faq

PRODUCTS WE SELL (EXACT LIST — DO NOT INVENT):
- APPLE: iPhone, iPad (A16, Air M4, Pro M5, Mini A17), MacBook (Air M5, Pro M5, Neo), iMac M4, Mac Mini M4, Mac Studio M4 Max, Apple Watch (SE 3, Series 11), iPad Pencil, Magic Keyboard, Studio Display.
- SAMSUNG: Galaxy phones only. NO Samsung tablets, NO Samsung watches.
- HONOR: Phones only. NO HONOR watches or tablets.
- XIAOMI/POCO/REDMI: Phones only.
- GOOGLE: Pixel phones only.
- OPPO, VIVO, INFINIX, ONEPLUS, NOTHING, REALME, RED MAGIC, ASUS, TECNO, iQOO: Phones only.
- REALME also has: Realme Pad 2 Lite (tablet).
- TECNO also has: TECNO MEGAPAD 11 (tablet).
- If a user asks for Samsung tablets, Samsung watches, or HONOR watches, say: "Sorry, we don't carry those. We only have Apple Watch for smartwatches and Apple iPad / Realme Pad / TECNO MEGAPAD for tablets."
- If a user asks for ANY brand NOT in the 15-brand list (Dell, Lenovo, Huawei, Motorola, Nokia, Sony, Acer, HP, etc.), say: "Sorry, we don't carry that brand."

REFUSAL RULE (CRITICAL — READ CAREFULLY):
- REAL IPHONE MODELS THAT EXIST: iPhone 12, 12 Mini, 12 Pro, 12 Pro Max, iPhone 13, 13 Mini, 13 Pro, 13 Pro Max, iPhone 14, 14 Plus, 14 Pro, 14 Pro Max, iPhone 15, 15 Plus, 15 Pro, 15 Pro Max, iPhone 16, 16 Plus, 16 Pro, 16 Pro Max, 16e, iPhone 17, 17e, 17 Pro, 17 Pro Max, 17 Air. ALL of these are REAL and AVAILABLE. NEVER say any of these don't exist.
- FICTIONAL models that DON'T exist: iPhone 18, iPhone 19, iPhone 99, or any number above 17. ONLY refuse these.
- REAL SAMSUNG models: Galaxy S22 through S26, Z Fold 6/7, Z Flip 7, A-series (A07 through A56). All real.
- Your training data may be outdated. ALWAYS trust the Context over your own knowledge. If a model appears in the Context catalog with pricing, it EXISTS. Do NOT contradict yourself by quoting a price and then later saying the model doesn't exist.
- You CANNOT write code. If asked for code, refuse in 1 sentence.

ELIGIBILITY HIGHLIGHTS (mention when relevant):
- We accept CTOS/CCRIS blacklists, self-employed, and loan rejects.
- Minimum age 18. Open to Government, Private, Self-Employed, and Gig workers.
- No documents needed to start. 100% approval focus.

POLICIES (mention when relevant):
- 2 years warranty, 1-2 business days delivery, free 90-day return.
- Nationwide delivery across Malaysia, 24x7 customer support.`;

// STRUCTURED INDEXING: Breaking the data into discrete chunks ensures
const KNOWLEDGE_DOCUMENTS = [
  {
    title: "GoAnsuran Core Business & Eligibility",
    content: `GoAnsuran (Mobile Wholesale City Malaysia) offers flexible installments. High Approval Rates for CTOS/CCRIS Blacklist and JCL/Aeon Credit rejects. Open to Government, Private, Self-Employed, and Gig Economy workers (minimum age 18).`
  },
  {
    title: "Business: GoAnsuran Overview & BNPL Services",
    content: `GoAnsuran is a Buy Now Pay Later (BNPL) provider in Malaysia. Tagline "Ansuran Ringan Tanpa Beban" means Lightweight / Burden-free Installment. "Tanpa Syarat Ketat" means No Strict Conditions. Nationwide delivery across Malaysia. 24x7 customer support center. Over 2,500 brands across 42 product categories. Covers mobile phones, computers, tablets, smart watches, cameras, home appliances, furniture, office equipment, smartphone repairs, and automotive parts. Instant Approval in Just 10 Minutes.`
  },
  {
    title: "Plan: GoFlexi (Rent-To-Own)",
    content: `GoFlexi is a Rent-To-Own plan with the HIGHEST approval rate. Suitable for: CTOS/CCRIS issues, rejected by JCL/AEON, irregular income — students, platform riders (Grab/foodpanda/ShopeeFood), self-employed, commission earners, foreign workers. How it works: pick a phone, pick a tenure, pay monthly and own it at the end. Tenures: 12, 18, 24, or 36 months. Processing time: approximately 1 day. Deposit: Required (low deposit, light installments). Pickup: in-branch or delivered. Apply online at https://goansuran.com/go-flexi. BM slogan: "Walau Blacklist 100% Dijamin Lulus" (Even if Blacklisted, 100% Guaranteed Approval).`
  },
  {
    title: "Plan: GoAngkasa (Civil Servant Salary Deduction)",
    content: `GoAngkasa is an EXCLUSIVE plan for civil servants (government servants with at least 6 months of service). Installments via ANGKASA automatic salary deduction (SPGA — Sistem Potongan Gaji ANGKASA). Secure and hassle-free. Tenures: 12 to 180 months (the longest tenure available, lowest monthly payments). Processing time: approximately 1 week. Deposit: Not required (RM 0). Includes phone + accessories + warranty package. PRICING: GoAngkasa has 12 different tenure options per model (12, 24, 36, 48, 60, 72, 84, 96, 108, 120, 144, 180 months). The full price list is too complex for chat. ALWAYS direct users to the GoAngkasa page for pricing: https://goansuran.com/go-angkasa. NEVER quote specific monthly amounts for GoAngkasa.`
  },
  {
    title: "Plan: JCL (Personal Loan)",
    content: `JCL is a Personal Loan option for customers with clean CTOS/CCRIS profiles and a stable source of income. Tenures: 12, 24, or 36 months. Processing time: 1 to 3 days. Deposit: Not required. Apply online at goansuran.com.`
  },
  {
    title: "Plan: BNPL (Buy Now Pay Later)",
    content: `BNPL (Buy Now Pay Later) is for instant checkout at walk-in branches using supported BNPL apps: Atome, AhaPay, or Grab. Tenures: 3 to 12 months (short tenures). Processing time: approximately 5 minutes (instant approval). Deposit: May be required. Available for walk-in customers only — not available online.`
  },
  {
    title: "Catalog: Supported Brands",
    content: `BRANDS WE SELL (COMPLETE LIST): Apple, Samsung, HONOR, Xiaomi, Google, OPPO, Vivo, Infinix, OnePlus, Nothing, Realme, Red Magic, Asus, TECNO, iQOO.

REAL IPHONE MODELS (ALL EXIST — NEVER SAY THEY DON'T): iPhone 12 (12, 12 Mini, 12 Pro, 12 Pro Max), iPhone 13 (13, 13 Mini, 13 Pro, 13 Pro Max), iPhone 14 (14, 14 Plus, 14 Pro, 14 Pro Max), iPhone 15 (15, 15 Plus, 15 Pro, 15 Pro Max), iPhone 16 (16, 16 Plus, 16 Pro, 16 Pro Max, 16e), iPhone 17 (17, 17e, 17 Pro, 17 Pro Max, 17 Air). iPhone 18 and above do NOT exist.

NON-PHONE PRODUCTS: Only Apple has non-phone devices — iPad, MacBook, iMac, Mac Mini, Mac Studio, Apple Watch, iPad Pencil, Magic Keyboard, Studio Display. Realme has Pad 2 Lite. TECNO has MEGAPAD 11. NO other brand has tablets or watches.

BRANDS WE DO NOT SELL: Dell, Lenovo, Huawei, Motorola, Nokia, Sony, Acer, HP, or any brand not listed above.`
  },
  {
    title: "Services: Device Repair",
    content: `We offer device repair services on installment, including: Express 30-Minute Screen Repairs, Screen Replacement, Battery Replacement, Back Glass Damage / Backglass Replacement, Charging Port Repair, and Rear Camera Lens repair. Covers both Apple and Android devices.`
  },
  {
    title: "Policies: Delivery, Warranty & Returns",
    content: `All products come with 2 years warranty. Delivery time is 1-2 business days. Free 90-day return policy. Nationwide delivery across all of Malaysia. 24x7 customer support. Example pricing: POCO X7 is RM1,199.00 MYR on installment.`
  },
  {
    title: "Business: Agent Recruitment",
    content: `GoAnsuran is actively recruiting agents. Marketing slogan: "Agent GoAnsuran Diperlukan Segera!" (GoAnsuran Agents Needed Urgently). Anyone can register as an agent, regardless of background. Agents help customers apply for installment plans and earn commissions. Direct interested users to the agent registration channel.`
  },
  {
    title: "Eligibility: Blacklist & Credit Policy",
    content: `GoAnsuran accepts CTOS/CCRIS blacklisted applicants. Marketing claim: "Walau Blacklist 100% Dijamin Lulus" (Even if Blacklisted, 100% Guaranteed Approval). "Tanpa Dokumen" (No Documents required). Open to Government, Private, Self-Employed, and Gig Economy workers (minimum age 18). Very high approval rates even for applicants previously rejected by JCL or Aeon Credit.`
  },
  {
    title: "Catalog: Other Products",
    content: `In addition to phones, GoAnsuran also offers: iPad tablets (Apple), smartwatches, accessories, and device repair services (screen, battery, backglass replacement on installment). We do NOT sell Dell, Lenovo, or any laptop/computer brand.`
  },
  {
    title: "Catalog: Supplementary Products",
    content: `GoAnsuran also offers device repair services on installment: Express 30-Minute Screen Repairs, Screen Replacement, Battery Replacement, Back Glass Damage / Backglass Replacement, Charging Port Repair, and Rear Camera Lens repair. Covers both Apple and Android devices from the 15 brands we carry.`
  },
  {
    title: "Application Process & Workflow",
    content: `HOW TO APPLY (the correct process — guide users through this): Step 1: Go to https://goansuran.com/auth/sign-in and register an account. Step 2: Complete the eKYC (electronic identity verification) — submit IC and basic info. Step 3: Once verified and approved, choose your device model, storage size, and installment plan (GoFlexi at https://goansuran.com/go-flexi, GoAngkasa at https://goansuran.com/go-angkasa, JCL, or BNPL). The chatbot CANNOT process applications, send emails, or collect personal data. The chatbot's job is to answer questions about products, plans, and eligibility, then direct users to https://goansuran.com to apply. Customer reviews and testimonials are available at https://goansuran.com/testimonial.`
  },
  {
    title: "Contact & Locations",
    content: `CONTACT DETAILS: Company is Global Bell Sdn Bhd (Company No. 832976-W), operating as GoAnsuran. HQ Address: Block B, Suites B607, Oasis Square, Jalan PJU 1A/7, Ara Damansara, 47301 Petaling Jaya, Selangor. Office Hours: Monday to Friday 9:00 AM - 6:00 PM. Phone/Call Support: 016-6666843. WhatsApp: 019-688 6440 (https://wa.me/60196886440). Email: enquiry@goansuran.com. BRANCHES & PICKUP LOCATIONS (all open daily, MWC Outlet & Pickup Point): 1) Ara Damansara - Oasis Square HQ (Level 6, B607). 2) Mid Valley Megamall (Level Penthouse, Centrepoint South). 3) Pavilion Kuala Lumpur (Level 16, Pavilion Tower KL). 4) One Utama Shopping Mall (Level 15, First Avenue Bandar Utama). 5) Sunway Square (Level 23, Corporate Tower 2). 6) 1 Mont Kiara (Level 13, Wisma Mont Kiara). 7) Bangsar South (Level 32, Tower B, The Vertical Corporate Towers). Customers can visit any branch for support, advice, or device pickup.`
  },
  {
    title: "Reviews & Testimonials",
    content: `CUSTOMER REVIEWS: Customers can read real reviews and testimonials from verified GoAnsuran buyers at https://goansuran.com/testimonial. GoAnsuran has served thousands of satisfied customers across Malaysia with flexible installment plans. The company offers 2 years warranty, 1-2 business days delivery, and a free 90-day return policy. When users ask about reviews, reputation, trust, or what other customers think, always direct them to https://goansuran.com/testimonial.`
  },
  {
    title: "FAQ: GoFlexi Program & Deposit",
    content: `FAQ — GOFLEXI PROGRAM: GoFlexi is a rent-to-own device program that lets you get a smartphone through affordable monthly payments. At the end of your contract, you can choose to return, own, or upgrade the device.

FAQ — DEPOSIT REQUIRED: Yes. Every GoFlexi plan requires a deposit. The deposit is NOT a fee — it remains yours. At the end of your contract (subject to minimum contract period and terms), you can choose to: (1) Return the phone and get your deposit back, (2) Use it toward owning the device, (3) Carry it forward when you upgrade. The deposit helps GoAnsuran approve more customers, including those who may not qualify under traditional financing.

FAQ — DEVICE CONDITION (IMPORTANT): For GoFlexi, devices are guaranteed to be in AS-NEW condition — fully functional, professionally tested, and in excellent physical condition. NOT brand new, but as-new. If the device doesn't meet this standard, report it within 24 hours of delivery for a free 1-to-1 replacement (typically within 3-7 working days). For all OTHER plans (GoAngkasa, JCL, BNPL, credit card installment), the devices are BRAND NEW.

FAQ — GOFLEXI PACKAGING: Each device comes with its box, charging cable, 20W power adapter, tempered-glass screen protector, and a protective case.

FAQ — GOFLEXI WARRANTY: Yes. All GoFlexi devices are covered by a GoAnsuran warranty for the duration of your contract (up to 3 years). Covers functional issues and manufacturing defects under normal use.`
  },
  {
    title: "FAQ: Application & Documents Required",
    content: `FAQ — HOW TO APPLY: Step 1: Submit your application at https://goansuran.com/auth/sign-in. Step 2: Once approved, complete your deposit payment and sign the electronic agreement in your user portal. Step 3: Receive your device by self-pickup or delivery.

FAQ — DOCUMENTS NEEDED: To process your application, prepare:
1. IC (NRIC).
2. Proof of income, based on your employment type:
   - SALARIED employee: latest 1 month payslip AND salaried bank statement.
   - GOVERNMENT servant: latest 3 months payslip.
   - SELF-EMPLOYED / business owner: latest 3 months company bank statements.
   - GIG / commission-based (Grab rider, foodpanda, etc): latest 1 month income or commission statements AND salaried bank statement.
   - OTHER: latest 3 months bank statements.
Submitting complete, clear documents helps review your application faster and improves approval chances.

FAQ — WHY DIFFERENT MODEL OFFERED: Your approved model is matched to your credit profile and affordability. After 6 months of good payment history, you may apply to upgrade to another model, subject to approval.

FAQ LINK: Full FAQ page is at https://goansuran.com/faq`
  }
 ];

async function seed() {
  console.log('====================================================');
  console.log('🤖 GoAnsuran RAG Studio - Live Data Ingestion');
  console.log('====================================================');

  try {
    console.log('\n🧹 Step 0: Clearing existing knowledge base...');
    const existingRes = await fetch(`${BACKEND_URL}/api/knowledge`);
    if (existingRes.ok) {
      const existingDocs = await existingRes.json();
      if (Array.isArray(existingDocs) && existingDocs.length > 0) {
        console.log(`   Found ${existingDocs.length} existing documents. Deleting...`);
        for (const doc of existingDocs) {
          await fetch(`${BACKEND_URL}/api/knowledge/${doc.id}`, { method: 'DELETE' });
        }
        console.log('   ✅ Cleared old data.');
      } else {
        console.log('   No existing documents found. Clean slate.');
      }
    }

    console.log('\n⚙️ Step 1: Seeding Studio Settings and Sales Prompt...');
    const settingsRes = await fetch(`${BACKEND_URL}/api/settings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        system_prompt: SYSTEM_PROMPT,
        ollama_model: 'google/gemini-3.1-flash-lite',
        ollama_embedding_model: 'nomic-embed-text:latest',
        rag_enabled: 'true'
      })
    });

    if (!settingsRes.ok) throw new Error(`Settings seed failed: ${await settingsRes.text()}`);
    console.log('✅ Settings seed successful! Anti-coding and out-of-scope defenses active.');

    console.log('\n📚 Step 2: Vectorizing GoAnsuran Knowledge Base in ChromaDB...');
    for (const doc of KNOWLEDGE_DOCUMENTS) {
      console.log(`   👉 Chunking and embedding: "${doc.title}"...`);
      const docRes = await fetch(`${BACKEND_URL}/api/knowledge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: doc.title,
          content: doc.content
        })
      });

      if (!docRes.ok) throw new Error(`Ingest failed for "${doc.title}": ${await docRes.text()}`);
      const data = await docRes.json();
      console.log(`      ✅ Indexed! Generated ${data.chunks || 1} semantic vectors.`);
    }

    console.log('\n====================================================');
    console.log('🎉 Seeding Completed Successfully! 🎉');
    console.log(`Ria is now loaded with ${KNOWLEDGE_DOCUMENTS.length} knowledge documents:`);
    console.log('- Expanded catalog covers 17 product categories (phones, Apple, motorcycles, appliances, etc.).');
    console.log('- iPhone 19 / fake device inquiries will trigger the strict refusal.');
    console.log('- Plans (GoFlexi, GoAngkasa, JCL, BNPL) are isolated for high-accuracy RAG search.');
    console.log('- Bahasa Malaysia slogans, delivery/warranty, and agent recruitment context indexed.');
    console.log('====================================================');

  } catch (error) {
    console.error('\n❌ Seeding Failed!');
    console.error(error.message);
  }
}

seed();
