/**
 * GoAnsuran RAG Studio - Automated Knowledge Seeder
 * 
 * This script seeds the GoAnsuran Sales Assistant rules, auto-enables RAG, 
 * and vectorizes complete brand guidelines, plans (GoFlexi, GoAngkasa), 
 * and catalog limits into ChromaDB.
 * 
 * Run command: node seed_goansuran.js
 */

const BACKEND_URL = 'http://localhost:3000';

const SYSTEM_PROMPT = `You are "Ria", the virtual Sales Assistant for GoAnsuran (goansuran.com). 
Your objective is to guide users to purchase smartphones or tablets on easy installment plans, qualify their eligibility, and lead them to submit an application.

CRITICAL RESPONDING STYLE:
- ALWAYS keep your responses extremely CONCISE, DIRECT, and SHORT.
- MAXIMUM length is 1 to 3 sentences per response. 
- NEVER write long paragraphs, bullet-point lists, or massive explanations.
- Keep text short to guarantee lightning-fast token generation speed!

CORE DIALOGUE FLOW:
1. QUALIFY ELIGIBILITY BRIEF:
   - Politely ask for: Their OCCUPATION (Gov servant vs Private sector) and MONTHLY INCOME.
   - Example: "Hi! I'd love to help you get the Samsung S24 on installments! Could you share what your current occupation is and your approximate monthly salary?"
2. RECOMMEND & VALIDATE:
   - Gov servant: Promote "GoAngkasa" (salary deduction, min RM 1,200).
   - Private / Self-employed: Promote "GoFlexi" (rent-to-own, min RM 1,500).
   - Example: "With your RM 2,000 salary in the private sector, you qualify perfectly for our GoFlexi Rent-to-Own plan! Let's get you set up on goansuran.com."
3. STRICT OUT-OF-SCOPE ENFORCEMENT:
   - If they ask about non-existent items (iPhone 19), unsupported catalog items (MacBook, laptops, TVs), or off-topic subjects (cooking, coding):
     Refuse immediately in exactly 1 brief sentence:
     "I can only help with GoAnsuran smartphone installment plans and cannot answer out-of-scope questions. What phone plan can I help you find today?"`;

const KNOWLEDGE_DOCUMENTS = [
  {
    title: "GoAnsuran Brand & Eligibility Guide",
    content: `GoAnsuran (operated in partnership with Mobile Wholesale City) is Malaysia's premier flexible smartphone installment platform. We make smartphones and tablets accessible to everyone through simplified approval processes, welcoming individuals who are blacklisted, have CCRIS/CTOS issues, or lack traditional credit cards.
    
    General eligibility criteria:
    - Must be a Malaysian citizen aged 18 to 60.
    - Requires a valid NRIC (MyKad).
    - Requires proof of income (latest 3 months payslips and 3 months bank statements).
    - No credit card required. In-house processing with high approval ratings.`
  },
  {
    title: "GoFlexi Installment Plan Details (Rent to Own)",
    content: `The GoFlexi plan (sometimes called GoFlexi Rent-to-Own) is designed for private sector employees, self-employed individuals, contract workers, and the general public.
    
    Core Features of GoFlexi:
    - Type: Rent-to-Own program. Pay easy monthly rentals, and ownership transfers to you at the end of the term.
    - Plan Terms available: 12 months, 18 months, or 24 months.
    - Minimum Monthly Salary Requirement: RM 1,500.
    - Required Documents: Copy of NRIC, latest 3 months payslips, and latest 3 months bank statements showing salary credit.
    - Open to self-employed individuals (requires business registration SSM and 6 months company bank statements).
    - Low upfront advance payment required upon approval.`
  },
  {
    title: "GoAngkasa Government Servant Plan Details (SPGA Deductions)",
    content: `The GoAngkasa plan is an exclusive installment scheme tailored specifically for government servants, public sector employees, and statutory body workers in Malaysia.
    
    Core Features of GoAngkasa:
    - Deductions: Integrated directly with SPGA (Sistem Potongan Gaji ANGKASA) for automated salary deduction.
    - Approval Rate: 99.9% approval rating.
    - CCRIS/CTOS: Fully friendly. Blacklisted government employees are highly accepted!
    - Minimum Monthly Salary Requirement: RM 1,200.
    - Required Documents: Copy of NRIC, latest 3 months payslips, and an ANGKASA deduction authorization form.
    - Advantages: Extremely low interest rates, zero credit card requirements, and highly secure payment deduction.`
  },
  {
    title: "GoAnsuran Product Catalog Limits & Rules",
    content: `GoAnsuran specializes strictly in smartphones, tablets, and mobile accessories.
    
    ACTIVE CATALOG PRODUCTS:
    - Apple: iPhone 17 Pro Max, iPhone 17 Pro, iPhone 17,iPhone 16 Pro, iPhone 16 Pro Max, iPhone 16, 5 Pro Max, iPhone 15 Pro, iPhone 15 Plus, iPhone 15, iPhone 14 Pro Max, iPhone 14, iPhone 13, iPhone 11, iPad Air, iPad Pro, and standard Apple iPads.
    - Samsung: Galaxy S24 Ultra, Galaxy S24+, Galaxy S24, Galaxy S23 Ultra, Galaxy Z Fold 5, Galaxy Z Flip 5, Galaxy A55, Galaxy A35, Galaxy Tab S9 series.
    - Other brands: Xiaomi, Oppo, Vivo, and Realme smartphones.
    
    PRODUCTS NOT SUPPORTED / OUT OF CATALOG:
    - Laptops (MacBook Air, MacBook Pro, ASUS, Dell, HP laptops are NOT sold by GoAnsuran).
    - Desktop computers (iMac, Mac Studio, custom PCs are NOT sold).
    - Smart Home appliances, TVs, or refrigerators are NOT in our catalog.
    - Future unreleased speculative devices (like "iPhone 19" or "iPhone 20" do not exist and are strictly out-of-scope).`
  }
];

async function seed() {
  console.log('====================================================');
  console.log('🤖 GoAnsuran RAG Studio - Auto Knowledge Ingestion');
  console.log('====================================================');
  console.log(`Connecting to Go backend at: ${BACKEND_URL}`);

  try {
    // 1. Set System prompt and settings in SQLite
    console.log('\n⚙️ Step 1: Seeding Studio Settings and Sales Prompt...');
    const settingsRes = await fetch(`${BACKEND_URL}/api/settings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        system_prompt: SYSTEM_PROMPT,
        ollama_model: 'llama3.2:latest',
        ollama_embedding_model: 'nomic-embed-text:latest',
        rag_enabled: 'true'
      })
    });

    if (!settingsRes.ok) {
      throw new Error(`Settings seed failed: ${await settingsRes.text()}`);
    }
    console.log('✅ Settings seed successful! RAG auto-enabled. System prompt set.');

    // 2. Vectorize documents in ChromaDB
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

      if (!docRes.ok) {
        throw new Error(`Ingest failed for "${doc.title}": ${await docRes.text()}`);
      }

      const data = await docRes.json();
      console.log(`      ✅ Indexed into ChromaDB! Generated ${data.chunks} semantic vectors.`);
    }

    console.log('\n====================================================');
    console.log('🎉 Seeding Completed Successfully! 🎉');
    console.log('====================================================');
    console.log('Ria is now fully trained on GoAnsuran products and plans!');
    console.log('Test parameters configured:');
    console.log(' - Lead Qualifications: Capture occupation & income');
    console.log(' - Scope restriction: Out-of-scope questions (MacBook, iPhone 19) will be refused.');
    console.log('====================================================');

  } catch (error) {
    console.error('\n❌ Seeding Failed!');
    console.error(error.message);
    console.log('\n👉 Note: Make sure the docker containers are running (`docker compose up -d`) before running this script.');
  }
}

seed();
