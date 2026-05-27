import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AnimatePresence, motion } from "framer-motion";
import {
  buildProjectPrompt,
  createChatSessionId,
  type ProjectAnswers,
} from "../api";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Answers = ProjectAnswers;

const EMPTY: Answers = {
  name: "",
  description: "",
  projectType: "",
  dataStorage: "",
  background: "",
  audience: "",
  usage: "",
  reliability: "",
  sensitiveData: "",
  location: "",
  domain: "",
};


const TEST_ANSWERS: Answers = {
  name: "My awesome project",
  description: "It helps users…",
  projectType: "A website people visit in a browser",
  dataStorage: "User accounts and records (names, orders, settings)",
  background: "Yes — things like sending emails or notifications",
  audience: "Our customers",
  usage: "Thousands",
  reliability: "It would be a serious problem",
  sensitiveData: "Yes",
  location: "North America",
  domain: "e.g. myapp.com",
};

// ---------------------------------------------------------------------------
// Step definitions
// ---------------------------------------------------------------------------

const TOTAL_STEPS = 9;

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function OptionCard({
  label,
  selected,
  onSelect,
}: {
  label: string;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`w-full rounded-xl border px-5 py-4 text-left text-sm transition-all ${
        selected
          ? "border-brand bg-brand text-white shadow-sm"
          : "border-border bg-surface-raised text-ink-secondary hover:border-border hover:bg-bg"
      }`}
    >
      {label}
    </button>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium text-ink-secondary">{label}</label>
      {children}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step renderers
// ---------------------------------------------------------------------------

function Step1({
  answers,
  set,
}: {
  answers: Answers;
  set: (k: keyof Answers, v: string) => void;
}) {
  return (
    <div className="flex flex-col gap-5">
      <Field label="What are we calling this project?">
        <input
          type="text"
          required
          value={answers.name}
          onChange={(e) => set("name", e.target.value)}
          placeholder="My awesome project"
          className="rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
        />
      </Field>
      <Field label="Describe what it does in a sentence or two">
        <textarea
          required
          rows={3}
          value={answers.description}
          onChange={(e) => set("description", e.target.value)}
          placeholder="It helps users…"
          className="resize-none rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
        />
      </Field>
    </div>
  );
}

function OptionStep({
  question,
  subheading,
  options,
  value,
  onChange,
}: {
  question: string;
  subheading?: string;
  options: string[];
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <p className="text-base font-medium text-ink">{question}</p>
        {subheading && (
          <p className="mt-1 text-sm text-ink-secondary">{subheading}</p>
        )}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        {options.map((opt) => (
          <OptionCard
            key={opt}
            label={opt}
            selected={value === opt}
            onSelect={() => onChange(opt)}
          />
        ))}
      </div>
    </div>
  );
}

function Step9({
  answers,
  set,
}: {
  answers: Answers;
  set: (k: keyof Answers, v: string) => void;
}) {
  const locationOptions = [
    "North America",
    "Europe",
    "Asia Pacific",
    "Spread across the world",
  ];
  return (
    <div className="flex flex-col gap-6">
      <OptionStep
        question="Where are most of your users located?"
        options={locationOptions}
        value={answers.location}
        onChange={(v) => set("location", v)}
      />
      <Field label="Got a website address in mind? (optional)">
        <input
          type="text"
          value={answers.domain}
          onChange={(e) => set("domain", e.target.value)}
          placeholder="e.g. myapp.com"
          className="rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-ink placeholder-ink-muted outline-none transition-colors focus:border-ink-muted focus:bg-surface-raised"
        />
      </Field>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Review screen
// ---------------------------------------------------------------------------

const REVIEW_LABELS: { label: string; key: keyof Answers }[] = [
  { label: "Project name", key: "name" },
  { label: "Description", key: "description" },
  { label: "Project type", key: "projectType" },
  { label: "Data storage", key: "dataStorage" },
  { label: "Background tasks", key: "background" },
  { label: "Audience", key: "audience" },
  { label: "Expected usage", key: "usage" },
  { label: "Reliability needs", key: "reliability" },
  { label: "Sensitive data", key: "sensitiveData" },
  { label: "User location", key: "location" },
  { label: "Domain (optional)", key: "domain" },
];

function ReviewScreen({
  answers,
  onBack,
  onSubmit,
  submitting,
  submitError,
}: {
  answers: Answers;
  onBack: () => void;
  onSubmit: () => void;
  submitting: boolean;
  submitError: string | null;
}) {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="text-xl font-semibold text-ink">
          Here's what we've got
        </h2>
        <p className="mt-1 text-sm text-ink-secondary">
          Review your answers before we get to work.
        </p>
      </div>

      <div className="rounded-xl border border-border bg-surface-raised divide-y divide-border-subtle">
        {REVIEW_LABELS.filter((r) => answers[r.key]).map((r) => (
          <div key={r.key} className="flex gap-4 px-5 py-3.5">
            <span className="w-40 shrink-0 text-xs font-medium text-ink-muted pt-0.5">
              {r.label}
            </span>
            <span className="text-sm text-ink-secondary">{answers[r.key]}</span>
          </div>
        ))}
      </div>

      {submitError && (
        <div className="rounded-xl bg-red-50 px-4 py-2.5 text-sm text-red-600">
          {submitError}
        </div>
      )}

      <div className="flex items-center justify-between pt-2">
        <button
          type="button"
          onClick={onBack}
          disabled={submitting}
          className="rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-ink-secondary transition-colors hover:bg-bg disabled:opacity-50"
        >
          Back
        </button>
        <button
          type="button"
          onClick={onSubmit}
          disabled={submitting}
          className="rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {submitting ? "Creating…" : "Create this project"}
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Slide animation variants
// ---------------------------------------------------------------------------

const variants = {
  enter: (dir: number) => ({
    x: dir > 0 ? 40 : -40,
    opacity: 0,
  }),
  center: { x: 0, opacity: 1 },
  exit: (dir: number) => ({
    x: dir > 0 ? -40 : 40,
    opacity: 0,
  }),
};

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function NewProjectPage() {
  const [step, setStep] = useState(1); // 1-9; 10 = review
  const [dir, setDir] = useState(1); // 1 = forward, -1 = back
  const [answers, setAnswers] = useState<Answers>(EMPTY);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);

  function set(key: keyof Answers, value: string) {
    setAnswers((prev) => ({ ...prev, [key]: value }));
  }

  function isStepValid(): boolean {
    switch (step) {
      case 1:
        return answers.name.trim() !== "" && answers.description.trim() !== "";
      case 2:
        return answers.projectType !== "";
      case 3:
        return answers.dataStorage !== "";
      case 4:
        return answers.background !== "";
      case 5:
        return answers.audience !== "";
      case 6:
        return answers.usage !== "";
      case 7:
        return answers.reliability !== "";
      case 8:
        return answers.sensitiveData !== "";
      case 9:
        return answers.location !== "";
      default:
        return true;
    }
  }

  function goNext() {
    if (!isStepValid()) return;
    setDir(1);
    setStep((s) => s + 1);
    containerRef.current?.scrollTo({ top: 0 });
  }

  function goBack() {
    setDir(-1);
    setStep((s) => Math.max(1, s - 1));
    containerRef.current?.scrollTo({ top: 0 });
  }



  async function handlesubmittest() {
    if (submitting) return;
    setSubmitError(null);
    setSubmitting(true);
    try {
      const sessionId = createChatSessionId();
      const initialPrompt = buildProjectPrompt(TEST_ANSWERS);
      sessionStorage.setItem(`gomcp-chat:${sessionId}`, initialPrompt);
      void navigate(`/projects/${sessionId}/chat`, {
        state: {
          sessionId,
          initialPrompt,
          projectName: TEST_ANSWERS.name,
        },
      });
    } catch (err: unknown) {
      setSubmitError(err instanceof Error ? err.message : "Something went wrong");
      setSubmitting(false);
    }

  }
  

  async function handleSubmit() {
    if (submitting) return;
    setSubmitError(null);
    setSubmitting(true);
    try {
      const sessionId = createChatSessionId();
      const initialPrompt = buildProjectPrompt(answers);
      sessionStorage.setItem(`gomcp-chat:${sessionId}`, initialPrompt);
      void navigate(`/projects/${sessionId}/chat`, {
        state: {
          sessionId,
          initialPrompt,
          projectName: answers.name,
        },
      });
    } catch (err: unknown) {
      setSubmitError(err instanceof Error ? err.message : "Something went wrong");
      setSubmitting(false);
    }
  }

  const progress = step <= TOTAL_STEPS ? (step - 1) / TOTAL_STEPS : 1;

  const stepTitles = [
    "Name & Description",
    "Project type",
    "Data storage",
    "Background tasks",
    "Audience",
    "Expected usage",
    "Reliability",
    "Sensitive data",
    "Location & domain",
  ];

  function renderStep() {
    switch (step) {
      case 1:
        return <Step1 answers={answers} set={set} />;
      case 2:
        return (
          <OptionStep
            question="Which of these best describes your project?"
            options={[
              "A website people visit in a browser",
              "A mobile app",
              "Something other apps or services connect to (an API)",
              "An automated process that runs in the background",
            ]}
            value={answers.projectType}
            onChange={(v) => set("projectType", v)}
          />
          
        );
      case 3:
        return (
          <OptionStep
            question="What kind of information does it need to remember?"
            options={[
              "Nothing — it doesn't store anything",
              "User accounts and records (names, orders, settings)",
              "Files, images, or videos",
              "Both records and files",
            ]}
            value={answers.dataStorage}
            onChange={(v) => set("dataStorage", v)}
          />
        );
      case 4:
        return (
          <OptionStep
            question="Will it ever need to work behind the scenes without someone actively using it?"
            options={[
              "No, it only does things when someone is actively using it",
              "Yes — things like sending emails or notifications",
              "Yes — things like processing uploads or generating reports",
              "Yes — it runs on a schedule (nightly, weekly, etc.)",
            ]}
            value={answers.background}
            onChange={(v) => set("background", v)}
          />
        );
      case 5:
        return (
          <OptionStep
            question="Who will be using this?"
            options={[
              "Just me",
              "My team (internal use only)",
              "Our customers",
              "Anyone — it will be public",
            ]}
            value={answers.audience}
            onChange={(v) => set("audience", v)}
          />
        );
      case 6:
        return (
          <OptionStep
            question="How busy do you expect it to get?"
            options={[
              "A small group — up to 50 or so",
              "Hundreds of users",
              "Thousands",
              "Tens of thousands",
            ]}
            value={answers.usage}
            onChange={(v) => set("usage", v)}
          />
        );
      case 7:
        return (
          <OptionStep
            question="What happens if it goes offline for a few minutes?"
            options={[
              "No big deal — it's not critical",
              "It would be inconvenient but okay",
              "It would be a serious problem",
              "It absolutely cannot go offline",
            ]}
            value={answers.reliability}
            onChange={(v) => set("reliability", v)}
          />
        );
      case 8:
        return (
          <OptionStep
            question="Will it store or handle sensitive information?"
            subheading="Things like passwords, payment details, or health data"
            options={["Yes", "No", "I'm not sure"]}
            value={answers.sensitiveData}
            onChange={(v) => set("sensitiveData", v)}
          />
        );
      case 9:
        return <Step9 answers={answers} set={set} />;
      default:
        return null;
    }
  }

  const isReview = step > TOTAL_STEPS;

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <button className="bg-brand text-white px-4 py-2 rounded hover:bg-brand-hover" onClick={handlesubmittest}>
        Submit Test Form
      </button>
      {/* Progress bar */}
      <div className="h-0.5 w-full bg-border-subtle">
        <motion.div
          className="h-full bg-brand"
          animate={{ width: `${progress * 100}%` }}
          transition={{ duration: 0.4, ease: "easeInOut" }}
        />
      </div>

      <div
        ref={containerRef}
        className="flex flex-1 flex-col items-center overflow-y-auto px-4 py-10"
      >
        <div className="w-full max-w-xl">
          {/* Step indicator */}
          {!isReview && (
            <div className="mb-6 flex items-center gap-3">
              <span className="text-xs font-medium text-ink-muted">
                Step {step} of {TOTAL_STEPS}
              </span>
              <span className="text-xs text-ink-muted">·</span>
              <span className="text-xs text-ink-secondary">
                {stepTitles[step - 1]}
              </span>
            </div>
          )}

          {/* Animated step content */}
          <AnimatePresence mode="wait" custom={dir}>
            <motion.div
              key={isReview ? "review" : step}
              custom={dir}
              variants={variants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={{ duration: 0.22, ease: "easeInOut" }}
            >
              {isReview ? (
                <ReviewScreen
                  answers={answers}
                  onBack={goBack}
                  onSubmit={() => void handleSubmit()}
                  submitting={submitting}
                  submitError={submitError}
                />
              ) : (
                <div className="flex flex-col gap-8">
                  {renderStep()}

                  {/* Navigation */}
                  <div className="flex items-center justify-between">
                    <button
                      type="button"
                      onClick={goBack}
                      disabled={step === 1}
                      className="rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-ink-secondary transition-colors hover:bg-bg disabled:pointer-events-none disabled:opacity-0"
                    >
                      Back
                    </button>
                    <button
                      type="button"
                      onClick={goNext}
                      disabled={!isStepValid()}
                      className="rounded-lg bg-brand px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-brand-hover disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      {step === TOTAL_STEPS ? "Review" : "Next"}
                    </button>
                  </div>
                </div>
              )}
            </motion.div>
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}

