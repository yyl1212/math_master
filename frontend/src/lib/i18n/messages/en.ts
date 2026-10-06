export const enMessages = {
  "study.checking": {"text": "Checking your study account…", "params": []},
  "study.state.unlearned": {"text": "Unlearned", "params": []},
  "study.state.learning": {"text": "Learning", "params": []},
  "study.state.completed": {"text": "Completed", "params": []},
  "study.state.reviewing": {"text": "Reviewing", "params": []},
  "study.complete": {"text": "Complete learning", "params": []},
  "study.review.start": {"text": "Start review", "params": []},
  "study.review.finish": {"text": "Finish review", "params": []},
  "study.retry": {"text": "Retry same request", "params": []},
  "study.saving": {"text": "Saving…", "params": []},
  "study.unavailable": {"text": "Study is temporarily unavailable.", "params": []},
  "study.uncertain": {"text": "No confirmation received. Retry the same request to confirm.", "params": []},
  "study.materialChanged": {"text": "The material has changed. Review the current version.", "params": []},
  "study.withdrawn": {"text": "This knowledge is unavailable. Your private records remain.", "params": []},
  "study.note.label": {"text": "Personal note", "params": []},
  "study.note.save": {"text": "Save note", "params": []},
  "study.note.delete": {"text": "Delete note", "params": []},
  "study.note.saved": {"text": "Note saved.", "params": []},
  "study.note.deleted": {"text": "Note deleted.", "params": []},
  "study.note.unsaved": {"text": "You have unsaved note edits.", "params": []},
  "study.note.latest": {"text": "Read latest note", "params": []},
  "study.note.useLatest": {"text": "Use latest note", "params": []},
  "study.note.leave": {"text": "Leave with unsaved note edits?", "params": []},
  "study.note.preview": {"text": "Note preview", "params": []},
  "study.note.conflict": {"text": "The note changed elsewhere. Read the latest revision; your edits are kept.", "params": []},
  "study.note.pendingEdits": {"text": "The earlier request is confirmed. Your new edits are still unsaved.", "params": []},
  "study.error.STUDY_INVALID": {"text": "Study input is invalid.", "params": []},
  "study.error.STUDY_NOT_CONFIGURED": {"text": "Study records are temporarily unavailable.", "params": []},
  "study.error.STUDY_STATE_CONFLICT": {"text": "Study state changed. Refresh before continuing.", "params": []},
  "study.error.STUDY_VERSION_STALE": {"text": "Study material changed. Refresh before continuing.", "params": []},
  "study.error.STUDY_IDEMPOTENCY_CONFLICT": {"text": "This request key was used for other input.", "params": []},
  "study.error.STUDY_NOTE_CONFLICT": {"text": "This note changed. Read the latest revision before saving.", "params": []},
  "study.error.REAUTH_REQUIRED": {"text": "Verify your password before continuing.", "params": []},
  "topic.map.intro": {"text": "Browse 63 main topics, 534 subtopics and 4,969 specific themes.", "params": []},
  "topic.map.search": {"text": "Search topics and knowledge", "params": []},
  "topic.map.level": {"text": "Topic level", "params": []},
  "topic.map.allLevels": {"text": "Automatic level", "params": []},
  "topic.map.level1": {"text": "Main topics", "params": []},
  "topic.map.level2": {"text": "Subtopics", "params": []},
  "topic.map.level3": {"text": "Specific themes", "params": []},
  "topic.map.kind": {"text": "Topic category", "params": []},
  "topic.map.primary": {"text": "Primary classification", "params": []},
  "topic.map.auxiliary": {"text": "Auxiliary categories", "params": []},
  "topic.map.other": {"text": "Other categories", "params": []},
  "topic.map.previous": {"text": "Previous topics", "params": []},
  "topic.map.next": {"text": "Next topics", "params": []},
  "topic.map.noResults": {"text": "No topics match your search.", "params": []},
  "topic.map.empty": {"text": "This topic has no published knowledge yet.", "params": []},
  "topic.map.children": {"text": "Topics to explore", "params": []},
  "topic.map.knowledge": {"text": "Knowledge to read", "params": []},
  "topic.map.previousKnowledge": {"text": "Previous knowledge", "params": []},
  "topic.map.nextKnowledge": {"text": "Next knowledge", "params": []},
  "topic.map.explore": {"text": "Explore knowledge topics", "params": []},
  "topic.map.note": {"text": "Classification entries describe the catalogue. Available knowledge is shown separately.", "params": []},
  "topic.alias.note": {"text": "This former domain spans the topics below. Exact knowledge membership follows its reviewed assignment.", "params": []},
  "page.topics.id": {"text": "Topic Details", "params": []},

  "topic.publication.title": {"text": "Paired knowledge and topic publication", "params": []},
  "topic.publication.version": {"text": "Taxonomy version SHA", "params": []},
  "topic.publication.submissions": {"text": "Approved submission IDs", "params": []},
  "topic.publication.prepare": {"text": "Prepare paired publication", "params": []},
  "topic.publication.activate": {"text": "Activate paired publication", "params": []},
  "topic.publication.failed": {"text": "Paired publication failed. Your input is preserved.", "params": []},
  "topic.publication.stale": {"text": "Published heads changed. Refresh and prepare a new pair.", "params": []},
  "topic.publication.prepared": {"text": "Pair prepared. Inspect topic changes before activation.", "params": []},
  "topic.publication.published": {"text": "Knowledge and topic heads activated together.", "params": []},
  "topic.publication.refresh": {"text": "Refresh published pair", "params": []},
  "topic.diff.added": {"text": "Added knowledge", "params": []},
  "topic.diff.removed": {"text": "Removed knowledge", "params": []},
  "topic.diff.changed": {"text": "Changed topic assignments", "params": []},

  "topic.assignment.title": {"text": "Knowledge topic assignment", "params": []},
  "topic.assignment.ids": {"text": "Specific topic IDs", "params": []},
  "topic.assignment.version": {"text": "Taxonomy version SHA", "params": []},
  "topic.assignment.batch": {"text": "Source batch SHA", "params": []},
  "topic.assignment.sources": {"text": "Exact source references (JSON)", "params": []},
  "topic.assignment.save": {"text": "Save topic assignment", "params": []},
  "topic.assignment.saved": {"text": "Topic assignment saved.", "params": []},
  "topic.assignment.failed": {"text": "Topic assignment was not saved. Keep your selections and try again.", "params": []},
  "topic.assignment.invalid": {"text": "Enter valid specific topics and exact captured source references.", "params": []},
  "topic.assignment.pending": {"text": "Topic assignment is incomplete; verify it before submitting.", "params": []},
  "topic.assignment.frozen": {"text": "Frozen topic assignment", "params": []},
  "topic.assignment.check": {"text": "Relationships review also checks the frozen topic assignments below.", "params": []},

  "nav.knowledgeMap": {
    "text": "Knowledge Map",
    "params": []
  },
  "common.itemNumber": {
    "text": "Item {number}",
    "params": [
      "number"
    ]
  },
  "common.namedItem": {
    "text": "Item: {name}",
    "params": [
      "name"
    ]
  },
  "common.unavailable": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "auth.error.forbidden": {
    "text": "You do not have permission.",
    "params": []
  },
  "auth.error.notFound": {
    "text": "Resource not found.",
    "params": []
  },
  "feedback.error.notFound": {
    "text": "This report is unavailable.",
    "params": []
  },
  "locale.label": {
    "text": "Interface language",
    "params": []
  },
  "locale.english": {
    "text": "English",
    "params": []
  },
  "locale.chinese": {
    "text": "中文",
    "params": []
  },
  "locale.selection": {
    "text": "Current language: {language}",
    "params": [
      "language"
    ]
  },
  "layout.skip.to.content.ac576a": {
    "text": "Skip to content",
    "params": []
  },
  "layout.a.world.of.ideas.84541f": {
    "text": "A world of ideas",
    "params": []
  },
  "layout.built.on.clear.thinking.and.connected.learning.9c141f": {
    "text": "Built on clear thinking and connected learning.",
    "params": []
  },
  "site-header.math.master.home.f407d7": {
    "text": "Math Master home",
    "params": []
  },
  "site-header.a.world.of.ideas.883eb5": {
    "text": "A WORLD OF IDEAS",
    "params": []
  },
  "site-header.main.navigation.eb3559": {
    "text": "Main navigation",
    "params": []
  },
  "site-header.learning.hub.0af988": {
    "text": "Learning Hub",
    "params": []
  },
  "site-header.learn.ce78af": {
    "text": "Learn",
    "params": []
  },
  "site-header.my.reports.cc6e3f": {
    "text": "My reports",
    "params": []
  },
  "site-header.feedback.review.19092b": {
    "text": "Feedback review",
    "params": []
  },
  "site-header.notifications.788011": {
    "text": "Notifications",
    "params": []
  },
  "site-header.corrections.review.fc4dc2": {
    "text": "Corrections review",
    "params": []
  },
  "site-header.website.feedback.aea375": {
    "text": "Website feedback",
    "params": []
  },
  "auth-status.checking.account.d18f41": {
    "text": "Checking account…",
    "params": []
  },
  "auth-status.accounts.unavailable.5b2487": {
    "text": "Accounts unavailable",
    "params": []
  },
  "auth-status.my.learning.c60fdf": {
    "text": "My learning",
    "params": []
  },
  "auth-status.learning.history.35b7a5": {
    "text": "Learning history",
    "params": []
  },
  "auth-status.edit.content.f57e8e": {
    "text": "Edit content",
    "params": []
  },
  "auth-status.review.content.9e6e6c": {
    "text": "Review content",
    "params": []
  },
  "auth-status.publish.content.35b610": {
    "text": "Publish content",
    "params": []
  },
  "auth-status.write.questions.4f6b81": {
    "text": "Write questions",
    "params": []
  },
  "auth-status.review.questions.e2fd5a": {
    "text": "Review questions",
    "params": []
  },
  "auth-status.publish.question.bank.545b0c": {
    "text": "Publish question bank",
    "params": []
  },
  "auth-status.manage.users.58606e": {
    "text": "Manage users",
    "params": []
  },
  "auth-status.sign.in.bfd402": {
    "text": "Sign in",
    "params": []
  },
  "content-state.try.again.d8b839": {
    "text": "Try again",
    "params": []
  },
  "content-state.explore.knowledge.7bd0d2": {
    "text": "Explore knowledge",
    "params": []
  },
  "learning-hub.curious.minds.connected.ideas.55351d": {
    "text": "CURIOUS MINDS, CONNECTED IDEAS",
    "params": []
  },
  "learning-hub.a.clear.path.through.983c9a": {
    "text": "A clear path through",
    "params": []
  },
  "learning-hub.mathematics.4d3cd6": {
    "text": "mathematics.",
    "params": []
  },
  "learning-hub.build.understanding.one.idea.at.a.time.explore.the.foundations.fo.88ab73": {
    "text": "Build understanding, one idea at a time. Explore the foundations, follow the connections, and find a direction that inspires you.",
    "params": []
  },
  "learning-hub.explore.knowledge.dfe344": {
    "text": "Explore knowledge ",
    "params": []
  },
  "learning-hub.continue.your.learning.f95b7e": {
    "text": "Continue your learning",
    "params": []
  },
  "learning-hub.from.first.principles.to.new.frontiers.15da58": {
    "text": "From first principles to new frontiers.",
    "params": []
  },
  "learning-hub.every.idea.has.a.connection.69c99c": {
    "text": "EVERY IDEA HAS A CONNECTION",
    "params": []
  },
  "learning-hub.explore.e6e7d0": {
    "text": "Explore.",
    "params": []
  },
  "learning-hub.understand.57e3c1": {
    "text": "Understand.",
    "params": []
  },
  "learning-hub.connect.23c10a": {
    "text": "Connect.",
    "params": []
  },
  "learning-hub.your.starting.point.7ae59f": {
    "text": "YOUR STARTING POINT",
    "params": []
  },
  "learning-hub.value.learning.domains.b173e5": {
    "text": "{v0} learning domains",
    "params": [
      "v0"
    ]
  },
  "learning-hub.one.connected.map.many.ways.to.learn.e5a2a7": {
    "text": "One connected map. Many ways to learn.",
    "params": []
  },
  "learning-hub.value.published.knowledgevaluevalue.2e25cf": {
    "text": "{v0} published knowledge{v1}{v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "learning-hub.understanding.comes.first.fc29d3": {
    "text": "Understanding comes first.",
    "params": []
  },
  "learning-hub.learning.domains.are.open.to.explore.mathematical.content.appears.a8c425": {
    "text": "Learning domains are open to explore. Mathematical content appears here after review, with its conditions, sources, and version.",
    "params": []
  },
  "knowledge-map.a.connected.world.of.mathematics.4cd45c": {
    "text": "A CONNECTED WORLD OF MATHEMATICS",
    "params": []
  },
  "knowledge-map.choose.a.domain.explore.its.ideas.then.follow.a.learning.path.529a69": {
    "text": "Choose a domain. Explore its ideas, then follow a learning path.",
    "params": []
  },
  "knowledge-map.search.learning.domains.13c880": {
    "text": "Search learning domains",
    "params": []
  },
  "knowledge-map.search.english.or.chinese.topics.c67f21": {
    "text": "Search English or Chinese topics",
    "params": []
  },
  "knowledge-map.content.status.5efa57": {
    "text": "Content status",
    "params": []
  },
  "knowledge-map.all.domains.0f97da": {
    "text": "All domains",
    "params": []
  },
  "knowledge-map.in.development.5259ae": {
    "text": "In development",
    "params": []
  },
  "knowledge-map.published.2ef42e": {
    "text": "Published",
    "params": []
  },
  "knowledge-map.search.49c266": {
    "text": "Search",
    "params": []
  },
  "knowledge-map.groups.for.learning.with.room.for.connections.8ec81a": {
    "text": "Groups for learning, with room for connections.",
    "params": []
  },
  "knowledge-map.these.domains.are.practical.learning.groups.connections.across.do.a8e178": {
    "text": "These domains are practical learning groups. Connections across domains are part of the journey.",
    "params": []
  },
  "domain-view.breadcrumb.2bd873": {
    "text": "Breadcrumb",
    "params": []
  },
  "domain-view.learning.domain.value.8bbbb2": {
    "text": "LEARNING DOMAIN {v0}",
    "params": [
      "v0"
    ]
  },
  "domain-view.topics.to.explore.773408": {
    "text": "Topics to explore",
    "params": []
  },
  "domain-view.learning.paths.5f23ab": {
    "text": "Learning paths",
    "params": []
  },
  "domain-view.version.value.d2b5e7": {
    "text": "Version {v0}",
    "params": [
      "v0"
    ]
  },
  "domain-view.learning.paths.are.in.development.a67b76": {
    "text": "Learning paths are in development.",
    "params": []
  },
  "domain-view.reviewed.content.will.appear.here.as.this.domain.grows.ad70f5": {
    "text": "Reviewed content will appear here as this domain grows.",
    "params": []
  },
  "domain-view.current.content.f491db": {
    "text": "CURRENT CONTENT",
    "params": []
  },
  "domain-view.published.knowledgevaluevalue.251575": {
    "text": "published knowledge{v0}{v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "domain-view.explore.the.topics.freely.learning.paths.connect.the.ideas.when.r.7e8cfb": {
    "text": "Explore the topics freely. Learning paths connect the ideas when reviewed content is available.",
    "params": []
  },
  "domain-view.related.domains.97e705": {
    "text": "Related domains",
    "params": []
  },
  "domain-view.explore.another.direction.these.links.describe.connections.rather.a7437c": {
    "text": "Explore another direction. These links describe connections, rather than prerequisites.",
    "params": []
  },
  "knowledge-view.read.the.idea.understand.its.conditions.and.follow.the.connection.466280": {
    "text": "Read the idea, understand its conditions, and follow the connections.",
    "params": []
  },
  "knowledge-view.on.this.page.b5658f": {
    "text": "On this page",
    "params": []
  },
  "knowledge-view.on.this.page.073c74": {
    "text": "ON THIS PAGE",
    "params": []
  },
  "knowledge-view.the.core.idea.6956da": {
    "text": "THE CORE IDEA",
    "params": []
  },
  "knowledge-view.core.statement.e943c8": {
    "text": "Core statement",
    "params": []
  },
  "knowledge-view.conditions.97d4be": {
    "text": "Conditions",
    "params": []
  },
  "knowledge-view.scope.system.6be795": {
    "text": "Scope & system",
    "params": []
  },
  "knowledge-view.mathematical.system.eec854": {
    "text": "Mathematical system",
    "params": []
  },
  "knowledge-view.learning.goals.9e640a": {
    "text": "Learning goals",
    "params": []
  },
  "knowledge-view.proof.7fbb3c": {
    "text": "Proof",
    "params": []
  },
  "knowledge-view.explanations.de30ae": {
    "text": "Explanations",
    "params": []
  },
  "knowledge-view.examples.e68ee0": {
    "text": "Examples",
    "params": []
  },
  "knowledge-view.counterexamples.1e428c": {
    "text": "Counterexamples",
    "params": []
  },
  "knowledge-view.connections.dc2731": {
    "text": "Connections",
    "params": []
  },
  "knowledge-view.sources.use.31f2b5": {
    "text": "Sources & use",
    "params": []
  },
  "knowledge-view.accessed.value.908096": {
    "text": "Accessed {v0}",
    "params": [
      "v0"
    ]
  },
  "knowledge-view.back.to.the.knowledge.map.8bf2a2": {
    "text": "← Back to the knowledge map",
    "params": []
  },
  "asset-image.illustration.is.temporarily.unavailable.b89403": {
    "text": "Illustration is temporarily unavailable.",
    "params": []
  },
  "path-view.learning.path.8c4705": {
    "text": "LEARNING PATH",
    "params": []
  },
  "path-view.value.knowledge.points.38db06": {
    "text": "{v0} knowledge points",
    "params": [
      "v0"
    ]
  },
  "path-view.read.freely.and.follow.the.prerequisites.to.connect.the.ideas.ab5874": {
    "text": "Read freely and follow the prerequisites to connect the ideas.",
    "params": []
  },
  "path-view.knowledge.prerequisites.5450ad": {
    "text": "Knowledge prerequisites",
    "params": []
  },
  "path-view.prerequisites.865514": {
    "text": "Prerequisites",
    "params": []
  },
  "path-view.vvalue.55a29f": {
    "text": "v{v0}",
    "params": [
      "v0"
    ]
  },
  "path-view.no.prior.knowledge.required.1da6b0": {
    "text": "No prior knowledge required.",
    "params": []
  },
  "path-view.connections.show.prerequisites.for.this.path.version.reading.a.kn.477376": {
    "text": "Connections show prerequisites for this path version. Reading a knowledge point does not create a learning or assessment record.",
    "params": []
  },
  "safe-markdown.illustration.is.not.available.ef0518": {
    "text": "Illustration is not available.",
    "params": []
  },
  "safe-markdown.formula.could.not.be.displayed.1952de": {
    "text": "Formula could not be displayed. ",
    "params": []
  },
  "public.state.empty": {
    "text": "The catalogue is being prepared.",
    "params": []
  },
  "public.state.noResults": {
    "text": "No domains match your search.",
    "params": []
  },
  "public.state.notFound": {
    "text": "This content is not available.",
    "params": []
  },
  "public.state.unavailable": {
    "text": "Content is temporarily unavailable.",
    "params": []
  },
  "public.help.empty": {
    "text": "Learning domains will appear here as the catalogue takes shape.",
    "params": []
  },
  "public.help.noResults": {
    "text": "Try another topic, or explore all learning domains.",
    "params": []
  },
  "public.help.notFound": {
    "text": "Explore the knowledge map to find available content.",
    "params": []
  },
  "public.help.unavailable": {
    "text": "Please try again in a moment.",
    "params": []
  },
  "public.publishedOne": {
    "text": "{count} published knowledge point",
    "params": [
      "count"
    ]
  },
  "public.publishedMany": {
    "text": "{count} published knowledge points",
    "params": [
      "count"
    ]
  },
  "public.domainOne": {
    "text": "{count} domain",
    "params": [
      "count"
    ]
  },
  "public.domainMany": {
    "text": "{count} domains",
    "params": [
      "count"
    ]
  },
  "page.account": {
    "text": "Your account",
    "params": []
  },
  "page.admin.publications.id": {
    "text": "Fixed publication",
    "params": []
  },
  "page.admin.publications": {
    "text": "Reviewed publications",
    "params": []
  },
  "page.admin.question-publications.id": {
    "text": "Fixed question snapshot",
    "params": []
  },
  "page.admin.question-publications": {
    "text": "Trusted question publications",
    "params": []
  },
  "page.admin.question-withdrawals": {
    "text": "Permanent question withdrawal",
    "params": []
  },
  "page.admin.users": {
    "text": "People & permissions",
    "params": []
  },
  "page.admin.withdrawals": {
    "text": "Withdraw fixed content",
    "params": []
  },
  "page.assessments.id": {
    "text": "Assessment",
    "params": []
  },
  "page.assessments.id.result": {
    "text": "Assessment result",
    "params": []
  },
  "page.corrections.id": {
    "text": "Your learning correction",
    "params": []
  },
  "page.domains.id": {
    "text": "Learning domain",
    "params": []
  },
  "page.editor.drafts.id": {
    "text": "Edit mathematical content",
    "params": []
  },
  "page.editor.drafts.id.preview": {
    "text": "Read knowledge draft",
    "params": []
  },
  "page.editor": {
    "text": "Content workspaces",
    "params": []
  },
  "page.editor.questions.drafts.id": {
    "text": "Edit exact questions",
    "params": []
  },
  "page.editor.questions": {
    "text": "Question workspaces",
    "params": []
  },
  "page.feedback.id": {
    "text": "Report details",
    "params": []
  },
  "page.feedback.new": {
    "text": "Report a problem",
    "params": []
  },
  "page.feedback": {
    "text": "My reports",
    "params": []
  },
  "page.knowledge.id": {
    "text": "Knowledge",
    "params": []
  },
  "page.knowledge": {
    "text": "Knowledge Map",
    "params": []
  },
  "page.learn": {
    "text": "My learning",
    "params": []
  },
  "page.learning-history": {
    "text": "Learning history",
    "params": []
  },
  "page.login": {
    "text": "Sign in",
    "params": []
  },
  "page.notifications": {
    "text": "Notifications",
    "params": []
  },
  "page.home": {
    "text": "A world of ideas",
    "params": []
  },
  "page.paths.id": {
    "text": "Learning path",
    "params": []
  },
  "page.practice.id": {
    "text": "Practice",
    "params": []
  },
  "page.register": {
    "text": "Create account",
    "params": []
  },
  "page.review.id": {
    "text": "Fixed submission review",
    "params": []
  },
  "page.review.corrections.id": {
    "text": "Correction case",
    "params": []
  },
  "page.review.corrections.id.plans.planId.version": {
    "text": "Correction plan and review",
    "params": []
  },
  "page.review.corrections": {
    "text": "Correction cases",
    "params": []
  },
  "page.review.feedback.id": {
    "text": "Report details",
    "params": []
  },
  "page.review.feedback": {
    "text": "Feedback review",
    "params": []
  },
  "page.review": {
    "text": "Independent review",
    "params": []
  },
  "page.review.questions.id": {
    "text": "Frozen question review",
    "params": []
  },
  "page.review.questions": {
    "text": "Independent question review",
    "params": []
  },
  "loading.loading.content.e2e410": {
    "text": "Loading content…",
    "params": []
  },
  "page.invalid.search.parameters.29e585": {
    "text": "Invalid search parameters.",
    "params": []
  },
  "page.please.shorten.your.search.or.reset.the.filters.bd6ccd": {
    "text": "Please shorten your search or reset the filters.",
    "params": []
  },
  "page.reset.search.92df98": {
    "text": "Reset search",
    "params": []
  },
  "page.lesson.feedback.14669c": {
    "text": "Lesson feedback",
    "params": []
  },
  "page.unit.c57306": {
    "text": "Unit ",
    "params": []
  },
  "page.illustration.0ffea7": {
    "text": "Illustration ",
    "params": []
  },
  "page.route.feedback.729fbd": {
    "text": "Route feedback",
    "params": []
  },
  "credentials-form.make.room.for.mathematics.56b94d": {
    "text": "Make room for mathematics.",
    "params": []
  },
  "credentials-form.welcome.back.dfdbca": {
    "text": "Welcome back.",
    "params": []
  },
  "credentials-form.create.an.account.to.begin.your.next.chapter.16d5e3": {
    "text": "Create an account to begin your next chapter.",
    "params": []
  },
  "credentials-form.sign.in.to.your.math.master.account.ced01b": {
    "text": "Sign in to your Math Master account.",
    "params": []
  },
  "credentials-form.from.first.principles.to.new.frontiers.explore.a.connected.world..c2f0a1": {
    "text": "From first principles to new frontiers, explore a connected world of mathematical ideas.",
    "params": []
  },
  "credentials-form.explore.the.knowledge.map.d12c1a": {
    "text": "Explore the Knowledge Map →",
    "params": []
  },
  "credentials-form.username.e3b89e": {
    "text": "Username",
    "params": []
  },
  "credentials-form.password.e7cf3e": {
    "text": "Password",
    "params": []
  },
  "credentials-form.15.128.characters.spaces.and.unicode.characters.are.welcome.66a06f": {
    "text": "15–128 characters. Spaces and Unicode characters are welcome.",
    "params": []
  },
  "credentials-form.please.wait.4660a9": {
    "text": "Please wait…",
    "params": []
  },
  "credentials-form.already.have.an.account.04a38d": {
    "text": "Already have an account? ",
    "params": []
  },
  "credentials-form.new.to.math.master.733bb8": {
    "text": "New to Math Master? ",
    "params": []
  },
  "admin-users.administration.bbdbcb": {
    "text": "ADMINISTRATION",
    "params": []
  },
  "admin-users.manage.roles.and.help.verified.account.owners.regain.access.3f1280": {
    "text": "Manage roles and help verified account owners regain access.",
    "params": []
  },
  "admin-users.search.usernames.adb993": {
    "text": "Search usernames",
    "params": []
  },
  "admin-users.account.9af211": {
    "text": "account",
    "params": []
  },
  "admin-users.accounts.bc62a3": {
    "text": "accounts",
    "params": []
  },
  "admin-users.accounts.8a7c8b": {
    "text": "Accounts",
    "params": []
  },
  "admin-users.password.change.required.77288b": {
    "text": "Password change required",
    "params": []
  },
  "admin-users.no.matching.accounts.277fda": {
    "text": "No matching accounts.",
    "params": []
  },
  "admin-users.previous.a57b08": {
    "text": "Previous",
    "params": []
  },
  "admin-users.next.1ff57a": {
    "text": "Next",
    "params": []
  },
  "admin-users.roles.c25337": {
    "text": "Roles",
    "params": []
  },
  "admin-users.reason.f81ab8": {
    "text": "Reason",
    "params": []
  },
  "admin-users.save.roles.1a0378": {
    "text": "Save roles",
    "params": []
  },
  "admin-users.reset.password.e0edfe": {
    "text": "Reset password",
    "params": []
  },
  "admin-users.verify.ownership.outside.this.website.before.resetting.an.account.0883de": {
    "text": "Verify ownership outside this website before resetting an account. The owner must change this temporary password after signing in.",
    "params": []
  },
  "admin-users.ownership.verification.7438d7": {
    "text": "Ownership verification",
    "params": []
  },
  "admin-users.temporary.password.b20862": {
    "text": "Temporary password",
    "params": []
  },
  "admin-users.the.reason.above.also.applies.to.this.reset.6ef8fa": {
    "text": "The reason above also applies to this reset.",
    "params": []
  },
  "admin-users.verify.your.password.0ed67a": {
    "text": "Verify your password",
    "params": []
  },
  "admin-users.verification.lasts.five.minutes.you.will.submit.your.change.separ.332ca5": {
    "text": "Verification lasts five minutes. You will submit your change separately.",
    "params": []
  },
  "admin-users.your.password.bbda70": {
    "text": "Your password",
    "params": []
  },
  "admin-users.verify.password.f226eb": {
    "text": "Verify password",
    "params": []
  },
  "admin-users.cancel.19766e": {
    "text": "Cancel",
    "params": []
  },
  "account-panel.your.account.cee7fd": {
    "text": "YOUR ACCOUNT",
    "params": []
  },
  "account-panel.change.your.password.to.continue.1c1b58": {
    "text": "Change your password to continue",
    "params": []
  },
  "account-panel.change.password.3f9c99": {
    "text": "Change password",
    "params": []
  },
  "account-panel.changing.your.password.signs.you.out.on.every.device.sign.in.agai.c2a7e1": {
    "text": "Changing your password signs you out on every device. Sign in again with your new password.",
    "params": []
  },
  "account-panel.current.password.72ed2b": {
    "text": "Current password",
    "params": []
  },
  "account-panel.new.password.3dd9df": {
    "text": "New password",
    "params": []
  },
  "account-panel.sign.out.48f0d3": {
    "text": "Sign out",
    "params": []
  },
  "account-panel.sign.out.everywhere.af180f": {
    "text": "Sign out everywhere",
    "params": []
  },
  "auth-state.view.account.407143": {
    "text": "View account",
    "params": []
  },
  "auth-state.clear.sign.in.cookie.b7ae51": {
    "text": "Clear sign-in cookie",
    "params": []
  },
  "auth-state.explore.mathematics.088a8c": {
    "text": "Explore mathematics",
    "params": []
  },
  "auth.roles": {
    "text": "Roles",
    "params": []
  },
  "auth.role.learner": {
    "text": "learner",
    "params": []
  },
  "auth.role.editor": {
    "text": "editor",
    "params": []
  },
  "auth.role.reviewer": {
    "text": "reviewer",
    "params": []
  },
  "auth.role.admin": {
    "text": "admin",
    "params": []
  },
  "auth.input.credentials": {
    "text": "Use a username of 3–32 letters, digits or underscores, and a password of 15–128 characters.",
    "params": []
  },
  "auth.input.password": {
    "text": "Passwords must contain 15–128 characters.",
    "params": []
  },
  "auth.input.admin": {
    "text": "Provide a reason of 10–1000 characters. A password reset also requires ownership verification and a password of 15–128 characters.",
    "params": []
  },
  "auth.saved": {
    "text": "Changes saved.",
    "params": []
  },
  "auth.verified": {
    "text": "Password verified. Submit your change again.",
    "params": []
  },
  "auth.state.anonymous": {
    "text": "Sign in to view your account",
    "params": []
  },
  "auth.state.invalidCookie": {
    "text": "Your sign-in cookie needs to be cleared.",
    "params": []
  },
  "field.add": {
    "text": "Add {label}",
    "params": [
      "label"
    ]
  },
  "field.remove": {
    "text": "Remove {label} {number}",
    "params": [
      "label",
      "number"
    ]
  },
  "field.numbered": {
    "text": "{label} {number}",
    "params": [
      "label",
      "number"
    ]
  },
  "field.id": {
    "text": "{label} ID",
    "params": [
      "label"
    ]
  },
  "field.version": {
    "text": "{label} version",
    "params": [
      "label"
    ]
  },
  "auth.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "auth.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "auth.error.INVALID_CREDENTIALS": {
    "text": "Invalid username or password.",
    "params": []
  },
  "auth.error.AUTHENTICATION_REQUIRED": {
    "text": "Please sign in to continue.",
    "params": []
  },
  "auth.error.CSRF_FAILED": {
    "text": "Request verification failed.",
    "params": []
  },
  "auth.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password to continue.",
    "params": []
  },
  "auth.error.METHOD_NOT_ALLOWED": {
    "text": "Method not allowed.",
    "params": []
  },
  "auth.error.USERNAME_UNAVAILABLE": {
    "text": "This username is unavailable.",
    "params": []
  },
  "auth.error.ALREADY_AUTHENTICATED": {
    "text": "Sign out before using another account.",
    "params": []
  },
  "auth.error.LAST_ADMIN_REQUIRED": {
    "text": "At least one administrator is required.",
    "params": []
  },
  "auth.error.REAUTHENTICATION_REQUIRED": {
    "text": "Verify your password before continuing.",
    "params": []
  },
  "auth.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "auth.error.AUTH_NOT_CONFIGURED": {
    "text": "Accounts are temporarily unavailable.",
    "params": []
  },
  "auth.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "draft-editor.content.workspace.af267d": {
    "text": "CONTENT WORKSPACE",
    "params": []
  },
  "draft-editor.revision.value.valuevalue.7bf409": {
    "text": "Revision {v0} · {v1}{v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "draft-editor.authors.value.catalogue.sha.valuevalue.bf61d0": {
    "text": "Authors: {v0} · Catalogue SHA: {v1}{v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "draft-editor.keep.package.and.member.versions.explicit.content.changes.require.b77a94": {
    "text": "Keep package and member versions explicit. Content changes require new immutable versions.",
    "params": []
  },
  "draft-editor.read.saved.draft.daf801": {
    "text": "Read saved draft",
    "params": []
  },
  "draft-editor.save.draft.3de100": {
    "text": "Save draft",
    "params": []
  },
  "draft-editor.check.readiness.1e2cac": {
    "text": "Check readiness",
    "params": []
  },
  "draft-editor.submit.for.review.40447e": {
    "text": "Submit for review",
    "params": []
  },
  "draft-editor.export.json.ff2565": {
    "text": "Export JSON",
    "params": []
  },
  "draft-editor.save.before.reading.checking.or.submitting.3f949b": {
    "text": "Save before reading, checking or submitting.",
    "params": []
  },
  "draft-editor.review.always.uses.a.frozen.copy.of.the.saved.revision.86413e": {
    "text": "Review always uses a frozen copy of the saved revision.",
    "params": []
  },
  "draft-editor.draft.fields.476e2c": {
    "text": "Draft fields",
    "params": []
  },
  "draft-editor.import.draftinput.json.5ab8cb": {
    "text": "Import DraftInput JSON",
    "params": []
  },
  "draft-editor.catalogue.version.0654f3": {
    "text": "Catalogue version",
    "params": []
  },
  "draft-editor.original.illustrations.558c7d": {
    "text": "Original illustrations",
    "params": []
  },
  "draft-editor.asset.d59386": {
    "text": "asset",
    "params": []
  },
  "draft-editor.asset.value.4a6e21": {
    "text": "Asset {v0}",
    "params": [
      "v0"
    ]
  },
  "draft-editor.asset.086710": {
    "text": "Asset ",
    "params": []
  },
  "draft-editor.svg.file.4b0cda": {
    "text": " SVG file",
    "params": []
  },
  "draft-editor.back.to.workspaces.174743": {
    "text": "Back to workspaces",
    "params": []
  },
  "content-preview.readiness.check.2f70ae": {
    "text": "Readiness check",
    "params": []
  },
  "content-preview.machine.checks.passed.reviewer.approval.is.still.required.184ba1": {
    "text": "Machine checks passed. Reviewer approval is still required.",
    "params": []
  },
  "content-preview.content.needs.work.before.review.bb2a94": {
    "text": "Content needs work before review.",
    "params": []
  },
  "content-preview.structural.issues.value.completeness.issues.value.human.checks.va.6fe5e4": {
    "text": "Structural issues: {v0} · Completeness issues: {v1} · Human checks: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "content-preview.showing.the.first.100.issues.be469a": {
    "text": "Showing the first 100 issues.",
    "params": []
  },
  "content-preview.safe.preview.eeb51b": {
    "text": "Safe preview",
    "params": []
  },
  "content-preview.save.new.illustrations.to.preview.their.bound.bytes.deb444": {
    "text": "Save new illustrations to preview their bound bytes.",
    "params": []
  },
  "content-preview.value.value.version.value.f7a6b1": {
    "text": "{v0} · {v1} · Version {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "content-preview.domains.value.topics.value.e9a533": {
    "text": "Domains: {v0} · Topics: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "content-preview.statement.6171b2": {
    "text": "Statement",
    "params": []
  },
  "content-preview.scope.b073f6": {
    "text": "Scope",
    "params": []
  },
  "content-preview.system.6725e7": {
    "text": "System",
    "params": []
  },
  "content-preview.learning.objectives.d38eb0": {
    "text": "Learning objectives",
    "params": []
  },
  "content-preview.no.additional.conditions.declared.c0637b": {
    "text": "No additional conditions declared.",
    "params": []
  },
  "content-preview.sources.caf85b": {
    "text": "Sources",
    "params": []
  },
  "content-preview.accessed.value.5dc1f8": {
    "text": "Accessed: {v0}",
    "params": [
      "v0"
    ]
  },
  "content-preview.no.sources.yet.810fee": {
    "text": "No sources yet.",
    "params": []
  },
  "content-preview.knowledge.relationships.242c96": {
    "text": "Knowledge relationships",
    "params": []
  },
  "content-preview.value.value.vvalue.afac6e": {
    "text": "{v0}: {v1} v{v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "content-preview.no.relationships.declared.4d1ca5": {
    "text": "No relationships declared.",
    "params": []
  },
  "content-preview.value.version.value.1d1973": {
    "text": "{v0} · Version {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "content-preview.knowledge.value.vvalue.assets.value.523132": {
    "text": "Knowledge: {v0} v{v1} · Assets: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "content-preview.value.vvalue.domains.value.206a60": {
    "text": "{v0} v{v1} · Domains: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "content-preview.value.vvalue.bde90b": {
    "text": "{v0} v{v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "content-preview.no.learning.paths.in.this.package.09d6d0": {
    "text": "No learning paths in this package.",
    "params": []
  },
  "diff-panel.added.value.replaced.value.removed.value.83f44a": {
    "text": "Added: {v0} · Replaced: {v1} · Removed: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "diff-panel.member.7c968f": {
    "text": "Member",
    "params": []
  },
  "diff-panel.before.9bb725": {
    "text": "Before",
    "params": []
  },
  "diff-panel.after.7b68fe": {
    "text": "After",
    "params": []
  },
  "source-fields.private.source.mapping.a8b054": {
    "text": "Private source mapping",
    "params": []
  },
  "source-fields.source.mapping.a69a8e": {
    "text": "source mapping",
    "params": []
  },
  "source-fields.source.mapping.value.830c78": {
    "text": "Source mapping {v0}",
    "params": [
      "v0"
    ]
  },
  "submission-list.independent.review.queue.2ca777": {
    "text": "Independent review queue",
    "params": []
  },
  "submission-list.submissions.541db6": {
    "text": "Submissions",
    "params": []
  },
  "submission-list.each.submission.fixes.the.entire.batch.sources.authors.and.illust.330fde": {
    "text": "Each submission fixes the entire batch, sources, authors and illustration bytes.",
    "params": []
  },
  "submission-list.submission.status.eeb12e": {
    "text": "Submission status",
    "params": []
  },
  "submission-list.value.revision.value.6706c2": {
    "text": "{v0} · Revision {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "submission-list.frozen.digest.value.bfb773": {
    "text": "Frozen digest: {v0}",
    "params": [
      "v0"
    ]
  },
  "submission-list.no.matching.submissions.0c6f73": {
    "text": "No matching submissions.",
    "params": []
  },
  "draft-list.authoring.67b82c": {
    "text": "AUTHORING",
    "params": []
  },
  "draft-list.original.content.begins.here.saving.and.independent.review.have.s.c2ee37": {
    "text": "Original content begins here. Saving and independent review have separate checks.",
    "params": []
  },
  "draft-list.revision.value.value.value.machine.issues.0d005f": {
    "text": "Revision {v0} · {v1} · {v2} machine issues",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "draft-list.no.workspaces.yet.ded050": {
    "text": "No workspaces yet.",
    "params": []
  },
  "draft-list.create.workspace.4b8922": {
    "text": "Create workspace",
    "params": []
  },
  "draft-list.new.package.afcdcc": {
    "text": "New package",
    "params": []
  },
  "draft-list.new.package.id.c42b51": {
    "text": "New package ID",
    "params": []
  },
  "draft-list.new.package.version.12a769": {
    "text": "New package version",
    "params": []
  },
  "draft-list.new.catalogue.version.a016bc": {
    "text": "New catalogue version",
    "params": []
  },
  "draft-list.choose.a.trusted.imported.catalogue.version.5f3f9e": {
    "text": "Choose a trusted imported catalogue version.",
    "params": []
  },
  "draft-list.create.draft.07d07a": {
    "text": "Create draft",
    "params": []
  },
  "draft-list.adopt.an.imported.package.70604f": {
    "text": "Adopt an imported package",
    "params": []
  },
  "draft-list.the.package.id.and.version.above.identify.the.immutable.imported..a9c4df": {
    "text": "The package ID and version above identify the immutable imported package.",
    "params": []
  },
  "draft-list.adoption.reason.e89c31": {
    "text": "Adoption reason",
    "params": []
  },
  "draft-list.adopt.package.60672c": {
    "text": "Adopt package",
    "params": []
  },
  "content-state.content.is.not.available.3b7767": {
    "text": "Content is not available.",
    "params": []
  },
  "content-state.content.management.is.temporarily.unavailable.0bcb14": {
    "text": "Content management is temporarily unavailable.",
    "params": []
  },
  "draft-reading-view.private.draft.preview.9b4ed0": {
    "text": "PRIVATE DRAFT PREVIEW",
    "params": []
  },
  "draft-reading-view.value.package.version.value.7321bb": {
    "text": "{v0} · Package version {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "draft-reading-view.saved.revision.value.status.value.c0ecaa": {
    "text": "Saved revision {v0} · Status: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "draft-reading-view.read.only.preview.of.the.saved.revision.unsaved.edits.are.not.inc.a4ec9b": {
    "text": "Read-only preview of the saved revision. Unsaved edits are not included. This preview does not establish mathematical approval or publication, and does not record learning progress.",
    "params": []
  },
  "draft-reading-view.back.to.workspace.0f0c0c": {
    "text": "Back to workspace",
    "params": []
  },
  "draft-reading-view.all.workspaces.415c8e": {
    "text": "All workspaces",
    "params": []
  },
  "draft-reading-view.browse.knowledge.7bae8a": {
    "text": "Browse knowledge",
    "params": []
  },
  "draft-reading-view.search.knowledge.points.aa0709": {
    "text": "Search knowledge points",
    "params": []
  },
  "draft-reading-view.english.id.caaeb8": {
    "text": "English / 中文 / ID",
    "params": []
  },
  "draft-reading-view.showing.value.of.value.knowledge.points.bcc8df": {
    "text": "Showing {v0} of {v1} knowledge points.",
    "params": [
      "v0",
      "v1"
    ]
  },
  "draft-reading-view.knowledge.points.da18c7": {
    "text": "Knowledge points",
    "params": []
  },
  "draft-reading-view.value.vvalue.value.a961d3": {
    "text": "{v0} · v{v1} · {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "draft-reading-view.no.knowledge.points.in.this.saved.draft.982d22": {
    "text": "No knowledge points in this saved draft.",
    "params": []
  },
  "draft-reading-view.no.matching.knowledge.points.try.a.different.title.or.id.81c59a": {
    "text": "No matching knowledge points. Try a different title or ID.",
    "params": []
  },
  "draft-reading-view.previous.knowledge.point.30303e": {
    "text": "Previous knowledge point",
    "params": []
  },
  "draft-reading-view.next.knowledge.point.9a368f": {
    "text": "Next knowledge point",
    "params": []
  },
  "draft-reading-view.feedback.on.this.preview.124f01": {
    "text": "Feedback on this preview",
    "params": []
  },
  "draft-reading-view.copy.this.reference.into.your.site.feedback.site.feedback.does.no.2e75aa": {
    "text": "Copy this reference into your site feedback. Site feedback does not replace independent mathematical review.",
    "params": []
  },
  "draft-reading-view.preview.reference.0fdc50": {
    "text": "Preview reference",
    "params": []
  },
  "draft-reading-view.give.site.feedback.26c82b": {
    "text": "Give site feedback",
    "params": []
  },
  "publication-panel.publication.72e134": {
    "text": "PUBLICATION",
    "params": []
  },
  "publication-panel.reviewed.publication.snapshots.8acf21": {
    "text": "Reviewed publication snapshots",
    "params": []
  },
  "publication-panel.no.content.has.been.published.yet.e8904f": {
    "text": "No content has been published yet.",
    "params": []
  },
  "publication-panel.withdraw.a.fixed.version.2cca43": {
    "text": "Withdraw a fixed version",
    "params": []
  },
  "publication-panel.refresh.current.head.a29f92": {
    "text": "Refresh current head",
    "params": []
  },
  "publication-panel.approved.submissions.86299b": {
    "text": "Approved submissions",
    "params": []
  },
  "publication-panel.loading.approved.batches.e29b0b": {
    "text": "Loading approved batches…",
    "params": []
  },
  "publication-panel.no.approved.submissions.fc2830": {
    "text": "No approved submissions.",
    "params": []
  },
  "publication-panel.v.e40f81": {
    "text": " v",
    "params": []
  },
  "publication-panel.review.record.8f17cd": {
    "text": "Review record",
    "params": []
  },
  "publication-panel.previous.approved.batches.ee33bf": {
    "text": "Previous approved batches",
    "params": []
  },
  "publication-panel.next.approved.batches.570a44": {
    "text": "Next approved batches",
    "params": []
  },
  "publication-panel.publication.reason.8ae504": {
    "text": "Publication reason",
    "params": []
  },
  "publication-panel.prepare.snapshot.93466e": {
    "text": "Prepare snapshot",
    "params": []
  },
  "publication-panel.snapshot.history.d81e02": {
    "text": "Snapshot history",
    "params": []
  },
  "publication-panel.selected.snapshot.b60236": {
    "text": "Selected snapshot",
    "params": []
  },
  "publication-panel.no.snapshots.yet.dce32a": {
    "text": "No snapshots yet.",
    "params": []
  },
  "publication-panel.previous.snapshots.1ead42": {
    "text": "Previous snapshots",
    "params": []
  },
  "publication-panel.next.snapshots.28f079": {
    "text": "Next snapshots",
    "params": []
  },
  "publication-panel.value.catalogue.version.value.9b204b": {
    "text": "{v0} · Catalogue version {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "publication-panel.manifest.sha.value.22188b": {
    "text": "Manifest SHA: {v0}",
    "params": [
      "v0"
    ]
  },
  "publication-panel.fixed.manifest.and.review.evidence.8aaddf": {
    "text": "Fixed manifest and review evidence",
    "params": []
  },
  "publication-panel.published.content.changed.prepare.a.new.snapshot.b8bb36": {
    "text": "Published content changed. Prepare a new snapshot.",
    "params": []
  },
  "publication-panel.activate.snapshot.bd0268": {
    "text": "Activate snapshot",
    "params": []
  },
  "package-fields.schema.version.value.cc82ae": {
    "text": "Schema version: {v0}",
    "params": [
      "v0"
    ]
  },
  "package-fields.package.id.7a35d0": {
    "text": "Package ID",
    "params": []
  },
  "package-fields.package.version.359c57": {
    "text": "Package version",
    "params": []
  },
  "package-fields.knowledge.e0f895": {
    "text": "knowledge",
    "params": []
  },
  "package-fields.learning.units.a95e2f": {
    "text": "Learning units",
    "params": []
  },
  "package-fields.unit.385cfd": {
    "text": "unit",
    "params": []
  },
  "package-fields.path.a0af9f": {
    "text": "path",
    "params": []
  },
  "withdrawal-panel.content.correction.c9be22": {
    "text": "CONTENT CORRECTION",
    "params": []
  },
  "withdrawal-panel.withdrawal.is.permanent.for.this.target.dependent.content.leaves..ba7e1a": {
    "text": "Withdrawal is permanent for this target. Dependent content leaves the current publication without changing its history.",
    "params": []
  },
  "withdrawal-panel.problem.version.ff5f6e": {
    "text": "Problem version",
    "params": []
  },
  "withdrawal-panel.published.member.2d0d52": {
    "text": "Published member",
    "params": []
  },
  "withdrawal-panel.choose.a.published.member.95163c": {
    "text": "Choose a published member",
    "params": []
  },
  "withdrawal-panel.exact.target.93f4b6": {
    "text": "Exact target",
    "params": []
  },
  "withdrawal-panel.target.kind.607b91": {
    "text": "Target kind",
    "params": []
  },
  "withdrawal-panel.target.svg.sha.7c995e": {
    "text": "Target SVG SHA",
    "params": []
  },
  "withdrawal-panel.target.id.32a090": {
    "text": "Target ID",
    "params": []
  },
  "withdrawal-panel.target.version.2acb4e": {
    "text": "Target version",
    "params": []
  },
  "withdrawal-panel.withdrawal.reason.6fb3ad": {
    "text": "Withdrawal reason",
    "params": []
  },
  "withdrawal-panel.preview.withdrawal.69c4c0": {
    "text": "Preview withdrawal",
    "params": []
  },
  "withdrawal-panel.withdrawal.preview.6e7687": {
    "text": "Withdrawal preview",
    "params": []
  },
  "withdrawal-panel.based.on.head.value.afc16c": {
    "text": "Based on head: {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.withdraw.version.8e7a90": {
    "text": "Withdraw version",
    "params": []
  },
  "withdrawal-panel.refresh.head.and.clear.preview.43cd61": {
    "text": "Refresh head and clear preview",
    "params": []
  },
  "withdrawal-panel.back.to.publications.ebb489": {
    "text": "Back to publications",
    "params": []
  },
  "command-controls.retry.sends.the.original.input.and.request.key.18c664": {
    "text": "Retry sends the original input and request key.",
    "params": []
  },
  "command-controls.retry.previous.request.34b089": {
    "text": "Retry previous request",
    "params": []
  },
  "command-controls.discard.pending.request.b43e94": {
    "text": "Discard pending request",
    "params": []
  },
  "command-controls.verification.lasts.five.minutes.submit.your.change.separately.9ee53e": {
    "text": "Verification lasts five minutes. Submit your change separately.",
    "params": []
  },
  "review-panel.fixed.submission.4d5920": {
    "text": "FIXED SUBMISSION",
    "params": []
  },
  "review-panel.value.frozen.revision.value.catalogue.version.value.7db94e": {
    "text": "{v0} · Frozen revision {v1} · Catalogue version {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "review-panel.frozen.digest.203dee": {
    "text": "Frozen digest: ",
    "params": []
  },
  "review-panel.catalogue.sha.90659b": {
    "text": "Catalogue SHA: ",
    "params": []
  },
  "review-panel.authors.6f5a07": {
    "text": "Authors: ",
    "params": []
  },
  "review-panel.verify.historical.authorship.before.approval.9994ab": {
    "text": "Verify historical authorship before approval.",
    "params": []
  },
  "review-panel.frozen.source.mapping.and.illustration.metadata.6b0aa5": {
    "text": "Frozen source mapping and illustration metadata",
    "params": []
  },
  "review-panel.administrator.self.review.decision.3f56c1": {
    "text": "Administrator self-review decision",
    "params": []
  },
  "review-panel.self.review.decision.fde99a": {
    "text": "Self-review decision",
    "params": []
  },
  "review-panel.final.review.decision.599ee2": {
    "text": "Final review decision",
    "params": []
  },
  "review-panel.reviewer.value.184bca": {
    "text": "Reviewer: {v0}",
    "params": [
      "v0"
    ]
  },
  "review-panel.authors.cannot.review.their.own.or.inherited.content.2a304e": {
    "text": "Authors cannot review their own or inherited content.",
    "params": []
  },
  "review-panel.administrator.self.review.c05816": {
    "text": "Administrator self-review",
    "params": []
  },
  "review-panel.this.approval.records.an.administrator.reviewing.their.own.conten.f9ff22": {
    "text": "This approval records an administrator reviewing their own content. It is not an independent mathematical review.",
    "params": []
  },
  "review-panel.complete.every.check.before.approval.d2af4e": {
    "text": "Complete every check before approval",
    "params": []
  },
  "review-panel.review.note.c2d4c5": {
    "text": "Review note",
    "params": []
  },
  "review-panel.approve.submission.bce550": {
    "text": "Approve submission",
    "params": []
  },
  "review-panel.return.for.changes.af8999": {
    "text": "Return for changes",
    "params": []
  },
  "review-panel.create.revision.workspace.bdecb5": {
    "text": "Create revision workspace",
    "params": []
  },
  "review-panel.back.to.submissions.0d99b1": {
    "text": "Back to submissions",
    "params": []
  },
  "draft-editor.draft.saved.check.readiness.before.submitting.2f6566": {
    "text": "Draft saved. Check readiness before submitting.",
    "params": []
  },
  "draft-editor.json.imported.save.this.draft.to.run.machine.checks.710f77": {
    "text": "JSON imported. Save this draft to run machine checks.",
    "params": []
  },
  "draft-editor.draftinput.exported.f36ce4": {
    "text": "DraftInput exported.",
    "params": []
  },
  "draft-editor.illustration.imported.use.new.owner.and.unit.versions.when.its.me.7446be": {
    "text": "Illustration imported. Use new owner and unit versions when its meaning changes.",
    "params": []
  },
  "draft-editor.readiness.check.complete.f8a29a": {
    "text": "Readiness check complete.",
    "params": []
  },
  "draft-editor.asset.value.value.bb5eb5": {
    "text": "Asset {v0} {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "draft-editor.asset.value.knowledge.5e272b": {
    "text": "Asset {v0} knowledge",
    "params": [
      "v0"
    ]
  },
  "source-fields.value.kind.033e78": {
    "text": "{v0} kind",
    "params": [
      "v0"
    ]
  },
  "source-fields.source.mapping.value.knowledge.49b15e": {
    "text": "Source mapping {v0} knowledge",
    "params": [
      "v0"
    ]
  },
  "source-fields.source.mapping.value.value.b90966": {
    "text": "Source mapping {v0} {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "path-fields.value.id.0ad111": {
    "text": "{v0} ID",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.version.f8a7ae": {
    "text": "{v0} version",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.title.ed4f63": {
    "text": "{v0} title",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.chinese.title.74d780": {
    "text": "{v0} Chinese title",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.domain.id.066d9c": {
    "text": "{v0} domain ID",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.node.ce3c13": {
    "text": "{v0} node",
    "params": [
      "v0"
    ]
  },
  "path-fields.value.node.value.5f4a60": {
    "text": "{v0} node {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "unit-fields.value.knowledge.f59083": {
    "text": "{v0} knowledge",
    "params": [
      "v0"
    ]
  },
  "unit-fields.value.angle.df71ec": {
    "text": "{v0} angle",
    "params": [
      "v0"
    ]
  },
  "unit-fields.value.angle.value.kind.c74f64": {
    "text": "{v0} angle {v1} kind",
    "params": [
      "v0",
      "v1"
    ]
  },
  "unit-fields.value.angle.value.body.e39e0c": {
    "text": "{v0} angle {v1} body",
    "params": [
      "v0",
      "v1"
    ]
  },
  "unit-fields.value.example.7998ee": {
    "text": "{v0} example",
    "params": [
      "v0"
    ]
  },
  "unit-fields.value.counterexample.c715cc": {
    "text": "{v0} counterexample",
    "params": [
      "v0"
    ]
  },
  "unit-fields.value.asset.id.8ccbdd": {
    "text": "{v0} asset ID",
    "params": [
      "v0"
    ]
  },
  "publication-panel.snapshot.prepared.inspect.the.fixed.difference.before.activation.13cb36": {
    "text": "Snapshot prepared. Inspect the fixed difference before activation.",
    "params": []
  },
  "publication-panel.snapshot.activated.current.head.refreshed.cd5ead": {
    "text": "Snapshot activated. Current head refreshed.",
    "params": []
  },
  "package-fields.knowledge.value.dcb3fa": {
    "text": "Knowledge {v0}",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.type.dcad78": {
    "text": "{v0} type",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.topic.id.722de7": {
    "text": "{v0} topic ID",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.objective.8deb98": {
    "text": "{v0} objective",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.condition.b7f86b": {
    "text": "{v0} condition",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.source.5a7bf3": {
    "text": "{v0} source",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.source.value.40de80": {
    "text": "{v0} source {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "package-fields.value.relation.5dd358": {
    "text": "{v0} relation",
    "params": [
      "v0"
    ]
  },
  "package-fields.value.relation.value.kind.5f8baa": {
    "text": "{v0} relation {v1} kind",
    "params": [
      "v0",
      "v1"
    ]
  },
  "package-fields.value.relation.value.target.4649cf": {
    "text": "{v0} relation {v1} target",
    "params": [
      "v0",
      "v1"
    ]
  },
  "package-fields.unit.value.4b7fc7": {
    "text": "Unit {v0}",
    "params": [
      "v0"
    ]
  },
  "package-fields.path.value.73d0e4": {
    "text": "Path {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.inspect.the.full.removal.before.withdrawing.518633": {
    "text": "Inspect the full removal before withdrawing.",
    "params": []
  },
  "withdrawal-panel.withdrawal.complete.current.head.refreshed.0518f3": {
    "text": "Withdrawal complete. Current head refreshed.",
    "params": []
  },
  "review-panel.review.responsibility.statement.0a0fb9": {
    "text": "Review responsibility statement",
    "params": []
  },
  "review-panel.independence.statement.3adc69": {
    "text": "Independence statement",
    "params": []
  },
  "review-panel.review.saved.640b4e": {
    "text": "Review saved.",
    "params": []
  },
  "content.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "content.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "content.error.AUTHENTICATION_REQUIRED": {
    "text": "Please sign in to continue.",
    "params": []
  },
  "content.error.CSRF_FAILED": {
    "text": "Request verification failed.",
    "params": []
  },
  "content.error.FORBIDDEN": {
    "text": "You do not have permission.",
    "params": []
  },
  "content.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password to continue.",
    "params": []
  },
  "content.error.NOT_FOUND": {
    "text": "Resource not found.",
    "params": []
  },
  "content.error.METHOD_NOT_ALLOWED": {
    "text": "Method not allowed.",
    "params": []
  },
  "content.error.DRAFT_CONFLICT": {
    "text": "This draft has changed. Reload it before continuing.",
    "params": []
  },
  "content.error.REVIEW_CONFLICT": {
    "text": "This submission already has a review decision.",
    "params": []
  },
  "content.error.IMMUTABLE_CONFLICT": {
    "text": "This version conflicts with saved content.",
    "params": []
  },
  "content.error.VERSION_CONFLICT": {
    "text": "This version conflicts with saved content.",
    "params": []
  },
  "content.error.IDEMPOTENCY_CONFLICT": {
    "text": "This request key was already used for different input.",
    "params": []
  },
  "content.error.PUBLICATION_STALE": {
    "text": "Published content has changed. Prepare a new snapshot.",
    "params": []
  },
  "content.error.PAYLOAD_TOO_LARGE": {
    "text": "This content exceeds the request size limit.",
    "params": []
  },
  "content.error.CONTENT_NOT_READY": {
    "text": "Complete the required content before submitting.",
    "params": []
  },
  "content.error.CONTENT_INVALID": {
    "text": "Content validation failed.",
    "params": []
  },
  "content.error.REVIEW_REQUIRED": {
    "text": "Independent review is required before publication.",
    "params": []
  },
  "content.error.CONTENT_LIMIT_EXCEEDED": {
    "text": "Split this content into smaller reviewed batches.",
    "params": []
  },
  "content.error.REAUTHENTICATION_REQUIRED": {
    "text": "Verify your password before continuing.",
    "params": []
  },
  "content.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "content.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "content.error.AUTH_NOT_CONFIGURED": {
    "text": "Accounts are temporarily unavailable.",
    "params": []
  },
  "content.error.CONTENT_NOT_CONFIGURED": {
    "text": "Content management is temporarily unavailable.",
    "params": []
  },
  "content.check.mathematics": {
    "text": "Mathematics",
    "params": []
  },
  "content.check.explanations": {
    "text": "Explanations",
    "params": []
  },
  "content.check.relationships": {
    "text": "Relationships",
    "params": []
  },
  "content.check.sources": {
    "text": "Sources",
    "params": []
  },
  "content.check.illustrations": {
    "text": "Illustrations",
    "params": []
  },
  "content.field.title": {
    "text": "title",
    "params": []
  },
  "content.field.titleZh": {
    "text": "Chinese title",
    "params": []
  },
  "content.field.statement": {
    "text": "statement",
    "params": []
  },
  "content.field.scope": {
    "text": "scope",
    "params": []
  },
  "content.field.system": {
    "text": "system",
    "params": []
  },
  "content.field.proof": {
    "text": "proof",
    "params": []
  },
  "content.field.author": {
    "text": "author",
    "params": []
  },
  "content.field.url": {
    "text": "url",
    "params": []
  },
  "content.field.accessedAt": {
    "text": "accessedAt",
    "params": []
  },
  "content.field.license": {
    "text": "license",
    "params": []
  },
  "content.field.attribution": {
    "text": "attribution",
    "params": []
  },
  "content.field.batchSha256": {
    "text": "batchSha256",
    "params": []
  },
  "content.field.relativePath": {
    "text": "relativePath",
    "params": []
  },
  "content.field.sha256": {
    "text": "sha256",
    "params": []
  },
  "content.field.legacyId": {
    "text": "legacyId",
    "params": []
  },
  "content.field.note": {
    "text": "note",
    "params": []
  },
  "content.field.id": {
    "text": "id",
    "params": []
  },
  "content.field.path": {
    "text": "path",
    "params": []
  },
  "field.named": {
    "text": "{label} {field}",
    "params": [
      "label",
      "field"
    ]
  },
  "content.enum.concept": {
    "text": "concept",
    "params": []
  },
  "content.enum.definition": {
    "text": "definition",
    "params": []
  },
  "content.enum.axiom": {
    "text": "axiom",
    "params": []
  },
  "content.enum.theorem": {
    "text": "theorem",
    "params": []
  },
  "content.enum.corollary": {
    "text": "corollary",
    "params": []
  },
  "content.enum.method": {
    "text": "method",
    "params": []
  },
  "content.enum.mathematical-thinking": {
    "text": "mathematical-thinking",
    "params": []
  },
  "content.enum.prerequisite": {
    "text": "prerequisite",
    "params": []
  },
  "content.enum.derivation": {
    "text": "derivation",
    "params": []
  },
  "content.enum.related": {
    "text": "related",
    "params": []
  },
  "content.enum.original": {
    "text": "original",
    "params": []
  },
  "content.enum.external": {
    "text": "external",
    "params": []
  },
  "content.enum.editing": {
    "text": "editing",
    "params": []
  },
  "content.enum.submitted": {
    "text": "submitted",
    "params": []
  },
  "content.enum.pending": {
    "text": "pending",
    "params": []
  },
  "content.enum.approved": {
    "text": "approved",
    "params": []
  },
  "content.enum.returned": {
    "text": "returned",
    "params": []
  },
  "content.enum.draft": {
    "text": "draft",
    "params": []
  },
  "content.enum.active": {
    "text": "active",
    "params": []
  },
  "content.enum.withdrawn": {
    "text": "withdrawn",
    "params": []
  },
  "content.enum.approve": {
    "text": "approve",
    "params": []
  },
  "content.enum.return": {
    "text": "return",
    "params": []
  },
  "content.enum.knowledge": {
    "text": "knowledge",
    "params": []
  },
  "content.enum.unit": {
    "text": "unit",
    "params": []
  },
  "content.enum.path": {
    "text": "path",
    "params": []
  },
  "content.enum.asset": {
    "text": "asset",
    "params": []
  },
  "content.error.TRANSFER_SVG_INVALID": {
    "text": "Choose a valid, original SVG illustration.",
    "params": []
  },
  "content.error.TRANSFER_JSON_INVALID": {
    "text": "The file must contain a valid DraftInput envelope.",
    "params": []
  },
  "content.error.TRANSFER_DIGEST_MISMATCH": {
    "text": "Illustration digest does not match.",
    "params": []
  },
  "content.error.TRANSFER_EXPORT_INVALID": {
    "text": "Complete the content fields before exporting.",
    "params": []
  },
  "content.error.TRANSFER_UNAVAILABLE": {
    "text": "Content import failed.",
    "params": []
  },
  "draft-editor.question.authoring.932915": {
    "text": "QUESTION AUTHORING",
    "params": []
  },
  "draft-editor.saved.revision.value.value.value.b3456a": {
    "text": "Saved revision {v0} · {v1} {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "draft-editor.saving.machine.validation.and.independent.approval.are.separate.s.d9d962": {
    "text": "Saving, machine validation and independent approval are separate steps.",
    "params": []
  },
  "draft-editor.owner.value.authors.value.2be751": {
    "text": "Owner: {v0} · Authors: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "draft-editor.verify.historical.authorship.and.original.sources.before.approval.248a00": {
    "text": "Verify historical authorship and original sources before approval.",
    "params": []
  },
  "draft-editor.this.workspace.is.read.only.for.this.account.or.has.already.been..be1962": {
    "text": "This workspace is read-only for this account or has already been submitted.",
    "params": []
  },
  "draft-editor.editable.question.package.e30c58": {
    "text": "Editable question package",
    "params": []
  },
  "draft-editor.finite.question.templates.3f552e": {
    "text": "Finite question templates",
    "params": []
  },
  "draft-editor.rational.arithmetic.rational.comparison.missing.operand.05a6d3": {
    "text": "Rational arithmetic · Rational comparison · Missing operand",
    "params": []
  },
  "draft-editor.template.5cde0f": {
    "text": "template",
    "params": []
  },
  "draft-editor.fixed.questions.67f48f": {
    "text": "Fixed questions",
    "params": []
  },
  "draft-editor.fixed.question.b25887": {
    "text": "fixed question",
    "params": []
  },
  "draft-editor.assessment.blueprints.fcd71c": {
    "text": "Assessment blueprints",
    "params": []
  },
  "draft-editor.blueprint.b1ece0": {
    "text": "blueprint",
    "params": []
  },
  "draft-editor.editable.json.and.source.mapping.ceb2f0": {
    "text": "Editable JSON and source mapping",
    "params": []
  },
  "draft-editor.import.and.export.contain.editable.question.data.and.source.mappi.9b4d59": {
    "text": "Import and export contain editable question data and source mapping. Original author and review evidence are retained by the server.",
    "params": []
  },
  "draft-editor.editable.question.json.ff0b13": {
    "text": "Editable question JSON",
    "params": []
  },
  "draft-editor.apply.editable.json.94e26f": {
    "text": "Apply editable JSON",
    "params": []
  },
  "draft-editor.copy.current.fields.to.json.7379fb": {
    "text": "Copy current fields to JSON",
    "params": []
  },
  "draft-editor.export.editable.json.65164d": {
    "text": "Export editable JSON",
    "params": []
  },
  "draft-editor.import.question.json.file.3dea76": {
    "text": "Import question JSON file",
    "params": []
  },
  "draft-editor.validate.saved.revision.0603de": {
    "text": "Validate saved revision",
    "params": []
  },
  "draft-editor.reload.saved.version.3c845b": {
    "text": "Reload saved version",
    "params": []
  },
  "draft-editor.open.frozen.submission.c1dfbc": {
    "text": "Open frozen submission",
    "params": []
  },
  "draft-editor.back.to.question.workspaces.7f7e32": {
    "text": "Back to question workspaces",
    "params": []
  },
  "diff-panel.before.a2bcd3": {
    "text": "Before: ",
    "params": []
  },
  "diff-panel.none.dc937b": {
    "text": "None",
    "params": []
  },
  "diff-panel.after.84507c": {
    "text": "After: ",
    "params": []
  },
  "diff-panel.fixed.mathematical.differences.15d782": {
    "text": "Fixed mathematical differences",
    "params": []
  },
  "diff-panel.loading.fixed.differences.90b335": {
    "text": "Loading fixed differences…",
    "params": []
  },
  "diff-panel.value.changes.offset.value.3a4d07": {
    "text": "{v0} changes · Offset {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "diff-panel.previous.changes.35f067": {
    "text": "Previous changes",
    "params": []
  },
  "diff-panel.next.changes.d0a346": {
    "text": "Next changes",
    "params": []
  },
  "diff-panel.fixed.members.and.approval.evidence.4d4bdb": {
    "text": "Fixed members and approval evidence",
    "params": []
  },
  "diff-panel.value.members.offset.value.approval.provenance.can.change.without.d13b68": {
    "text": "{v0} members · Offset {v1}. Approval provenance can change without a mathematical replacement.",
    "params": [
      "v0",
      "v1"
    ]
  },
  "diff-panel.value.value.vvalue.610089": {
    "text": "{v0} · {v1} v{v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "diff-panel.sha.483ee8": {
    "text": "SHA ",
    "params": []
  },
  "diff-panel.package.98d9ee": {
    "text": "Package ",
    "params": []
  },
  "diff-panel.submission.fb1f17": {
    "text": "Submission ",
    "params": []
  },
  "diff-panel.decision.f78582": {
    "text": "Decision ",
    "params": []
  },
  "diff-panel.frozen.digest.35e32d": {
    "text": "Frozen digest ",
    "params": []
  },
  "diff-panel.inherited.from.5be696": {
    "text": "Inherited from ",
    "params": []
  },
  "diff-panel.previous.members.9423b1": {
    "text": "Previous members",
    "params": []
  },
  "diff-panel.next.members.5909b2": {
    "text": "Next members",
    "params": []
  },
  "submission-list.independent.question.review.queue.006495": {
    "text": "Independent question review queue",
    "params": []
  },
  "submission-list.frozen.question.submissions.250eef": {
    "text": "Frozen question submissions",
    "params": []
  },
  "submission-list.review.uses.immutable.submitted.questions.and.every.bound.instanc.50a797": {
    "text": "Review uses immutable submitted questions and every bound instance. Authors are excluded from the independent queue.",
    "params": []
  },
  "submission-list.status.8bdde2": {
    "text": "Status ",
    "params": []
  },
  "submission-list.pending.331551": {
    "text": "Pending",
    "params": []
  },
  "submission-list.approved.87b42e": {
    "text": "Approved",
    "params": []
  },
  "submission-list.returned.361023": {
    "text": "Returned",
    "params": []
  },
  "submission-list.value.frozen.revision.value.cfefdf": {
    "text": "{v0} · Frozen revision {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "submission-list.digest.value.b74ba8": {
    "text": "Digest: {v0}",
    "params": [
      "v0"
    ]
  },
  "submission-list.no.submissions.in.this.view.28cce2": {
    "text": "No submissions in this view.",
    "params": []
  },
  "submission-list.previous.submissions.8d9ab0": {
    "text": "Previous submissions",
    "params": []
  },
  "submission-list.next.submissions.ea5832": {
    "text": "Next submissions",
    "params": []
  },
  "question-state.question.data.is.not.available.fa945f": {
    "text": "Question data is not available.",
    "params": []
  },
  "question-state.the.trusted.question.bank.is.not.configured.yet.3d8238": {
    "text": "The trusted question bank is not configured yet.",
    "params": []
  },
  "question-state.question.management.is.temporarily.unavailable.53681c": {
    "text": "Question management is temporarily unavailable.",
    "params": []
  },
  "draft-list.trusted.question.bank.a62caf": {
    "text": "TRUSTED QUESTION BANK",
    "params": []
  },
  "draft-list.build.exact.questions.validate.the.finite.batch.and.submit.it.for.03a844": {
    "text": "Build exact questions, validate the finite batch and submit it for independent review.",
    "params": []
  },
  "draft-list.no.question.workspaces.yet.84226e": {
    "text": "No question workspaces yet.",
    "params": []
  },
  "draft-list.previous.workspaces.fa8a2d": {
    "text": "Previous workspaces",
    "params": []
  },
  "draft-list.next.workspaces.e03049": {
    "text": "Next workspaces",
    "params": []
  },
  "draft-list.create.question.workspace.2b8d7c": {
    "text": "Create question workspace",
    "params": []
  },
  "draft-list.new.editable.package.9d8c64": {
    "text": "New editable package",
    "params": []
  },
  "draft-list.new.question.package.id.fc85e8": {
    "text": "New question package ID",
    "params": []
  },
  "draft-list.new.question.package.version.6b2659": {
    "text": "New question package version",
    "params": []
  },
  "draft-list.adopt.an.imported.question.package.35dce8": {
    "text": "Adopt an imported question package",
    "params": []
  },
  "draft-list.use.the.exact.package.id.and.version.above.historical.authorship..5ae8a4": {
    "text": "Use the exact package ID and version above. Historical authorship is retained; independent approval is still required.",
    "params": []
  },
  "draft-list.editor.permission.is.required.to.create.or.change.questions.0931c2": {
    "text": "Editor permission is required to create or change questions.",
    "params": []
  },
  "template-fields.objective.index.eab754": {
    "text": " objective index ",
    "params": []
  },
  "template-fields.value.coverage.value.f228fd": {
    "text": "{v0} coverage {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.load.published.objectives.a9108a": {
    "text": "Load published objectives",
    "params": []
  },
  "template-fields.value.finite.exact.parameters.572b12": {
    "text": "{v0} · finite exact parameters",
    "params": [
      "v0"
    ]
  },
  "template-fields.answer.placeholders.belong.in.the.explanation.values.remain.exact.c1d000": {
    "text": "Answer placeholders belong in the explanation. Values remain exact text; Go generates and independently verifies the finite batch.",
    "params": []
  },
  "template-fields.generator.version.value.verifier.version.value.bda518": {
    "text": "Generator version {v0} · Verifier version {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "coverage-panel.published.question.coverage.567402": {
    "text": "Published question coverage",
    "params": []
  },
  "coverage-panel.counts.reflect.approved.published.questions.whose.current.knowled.630f4d": {
    "text": "Counts reflect approved, published questions whose current knowledge, units and illustrations remain available.",
    "params": []
  },
  "coverage-panel.knowledge.head.b7020e": {
    "text": "Knowledge head: ",
    "params": []
  },
  "coverage-panel.question.head.6001f1": {
    "text": "Question head: ",
    "params": []
  },
  "coverage-panel.snapshots.changed.refresh.from.the.first.page.before.comparing.co.a168f3": {
    "text": "Snapshots changed. Refresh from the first page before comparing coverage.",
    "params": []
  },
  "coverage-panel.published.trusted.instances.value.0943fe": {
    "text": "Published trusted instances: {v0}",
    "params": [
      "v0"
    ]
  },
  "coverage-panel.approved.templates.value.fixed.questions.value.published.knowledg.47b5a4": {
    "text": "Approved templates: {v0} · Fixed questions: {v1} · Published knowledge nodes: {v2} · Duplicate occurrences excluded: {v3}",
    "params": [
      "v0",
      "v1",
      "v2",
      "v3"
    ]
  },
  "coverage-panel.node.counts.may.overlap.global.counts.use.distinct.question.ident.746aef": {
    "text": "Node counts may overlap; global counts use distinct question identities.",
    "params": []
  },
  "coverage-panel.ready.for.five.questions.16340b": {
    "text": "Ready for five questions",
    "params": []
  },
  "coverage-panel.not.ready.for.assessment.386442": {
    "text": "Not ready for assessment",
    "params": []
  },
  "coverage-panel.no.assessment.blueprint.configured.723360": {
    "text": "No assessment blueprint configured.",
    "params": []
  },
  "coverage-panel.valid.questions.value.fixed.value.generated.value.blueprint.pool..dd1cd1": {
    "text": "Valid questions: {v0} · Fixed: {v1} · Generated: {v2} · Blueprint pool: {v3}",
    "params": [
      "v0",
      "v1",
      "v2",
      "v3"
    ]
  },
  "coverage-panel.core.objectives.value.covered.value.e199da": {
    "text": "Core objectives: {v0} · Covered: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "coverage-panel.supplementary.objectives.value.3fa75c": {
    "text": "Supplementary objectives: {v0}",
    "params": [
      "v0"
    ]
  },
  "coverage-panel.value.coverage.rows.offset.value.d50be7": {
    "text": "{v0} coverage rows · Offset {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "coverage-panel.refresh.coverage.7fbd1b": {
    "text": "Refresh coverage",
    "params": []
  },
  "coverage-panel.previous.coverage.nodes.67c2fb": {
    "text": "Previous coverage nodes",
    "params": []
  },
  "coverage-panel.next.coverage.nodes.dff886": {
    "text": "Next coverage nodes",
    "params": []
  },
  "publication-panel.question.publication.ef452a": {
    "text": "QUESTION PUBLICATION",
    "params": []
  },
  "publication-panel.trusted.question.snapshots.dca705": {
    "text": "Trusted question snapshots",
    "params": []
  },
  "publication-panel.current.knowledge.head.1a83a3": {
    "text": "Current knowledge head: ",
    "params": []
  },
  "publication-panel.current.question.head.51281d": {
    "text": "Current question head: ",
    "params": []
  },
  "publication-panel.current.published.bank.value.templates.value.trusted.instances.va.30647f": {
    "text": "Current published bank: {v0} templates · {v1} trusted instances · {v2} blueprints.",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "publication-panel.withdraw.a.question.version.permanently.114e83": {
    "text": "Withdraw a question version permanently",
    "params": []
  },
  "publication-panel.refresh.both.current.heads.13c170": {
    "text": "Refresh both current heads",
    "params": []
  },
  "publication-panel.select.1.20.independently.approved.submissions.inherited.members..a32964": {
    "text": "Select 1–20 independently approved submissions. Inherited members retain their fixed evidence.",
    "params": []
  },
  "publication-panel.loading.approved.submissions.523773": {
    "text": "Loading approved submissions…",
    "params": []
  },
  "publication-panel.frozen.review.583f9b": {
    "text": "Frozen review",
    "params": []
  },
  "publication-panel.previous.approved.submissions.75d73f": {
    "text": "Previous approved submissions",
    "params": []
  },
  "publication-panel.next.approved.submissions.d59bee": {
    "text": "Next approved submissions",
    "params": []
  },
  "publication-panel.value.submissions.selected.across.pages.f7aa8e": {
    "text": "{v0} submissions selected across pages.",
    "params": [
      "v0"
    ]
  },
  "publication-panel.no.snapshots.on.this.page.2f19fb": {
    "text": "No snapshots on this page.",
    "params": []
  },
  "publication-panel.open.fixed.snapshot.d13edd": {
    "text": "Open fixed snapshot",
    "params": []
  },
  "publication-panel.selected.value.snapshot.978a63": {
    "text": "Selected {v0} snapshot",
    "params": [
      "v0"
    ]
  },
  "publication-panel.manifest.sha.aafd9c": {
    "text": "Manifest SHA: ",
    "params": []
  },
  "publication-panel.base.knowledge.head.da6971": {
    "text": "Base knowledge head: ",
    "params": []
  },
  "publication-panel.base.question.head.c7e74a": {
    "text": "Base question head: ",
    "params": []
  },
  "publication-panel.value.templates.value.instances.value.blueprints.d899e0": {
    "text": "{v0} templates · {v1} instances · {v2} blueprints",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "publication-panel.published.snapshots.changed.refresh.both.heads.and.prepare.a.new..126274": {
    "text": "Published snapshots changed. Refresh both heads and prepare a new snapshot.",
    "params": []
  },
  "withdrawal-panel.permanent.question.withdrawal.33c686": {
    "text": "PERMANENT QUESTION WITHDRAWAL",
    "params": []
  },
  "withdrawal-panel.withdraw.an.exact.question.version.d8e270": {
    "text": "Withdraw an exact question version",
    "params": []
  },
  "withdrawal-panel.template.withdrawal.removes.its.generated.instances.and.dependent.865351": {
    "text": "Template withdrawal removes its generated instances and dependent blueprints. Fixed-instance withdrawal removes explicit dependent blueprints. A single generated-instance withdrawal removes that instance; remaining pools are recalculated.",
    "params": []
  },
  "withdrawal-panel.a.node.may.no.longer.have.five.valid.questions.after.withdrawal.p.54a61b": {
    "text": "A node may no longer have five valid questions after withdrawal. Published explanations remain available. Historical bodies and approval records are preserved; withdrawn versions cannot be restored by republishing old approvals.",
    "params": []
  },
  "withdrawal-panel.current.question.head.value.2cd82b": {
    "text": "Current question head: {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.exact.version.to.withdraw.f7625e": {
    "text": "Exact version to withdraw",
    "params": []
  },
  "withdrawal-panel.you.may.identify.a.historical.version.that.is.no.longer.in.the.cu.67dc63": {
    "text": "You may identify a historical version that is no longer in the current bank.",
    "params": []
  },
  "withdrawal-panel.browse.current.members.944018": {
    "text": "Browse current members",
    "params": []
  },
  "withdrawal-panel.current.fixed.members.2494e0": {
    "text": "Current fixed members",
    "params": []
  },
  "withdrawal-panel.previous.current.members.eb0309": {
    "text": "Previous current members",
    "params": []
  },
  "withdrawal-panel.next.current.members.9edb05": {
    "text": "Next current members",
    "params": []
  },
  "withdrawal-panel.complete.impact.totals.a7beee": {
    "text": "Complete impact totals",
    "params": []
  },
  "withdrawal-panel.affected.templates.value.9c33ae": {
    "text": "Affected templates: {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.affected.instances.value.d7268d": {
    "text": "Affected instances: {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.affected.blueprints.value.30f428": {
    "text": "Affected blueprints: {v0}",
    "params": [
      "v0"
    ]
  },
  "withdrawal-panel.impact.digest.c7887e": {
    "text": "Impact digest: ",
    "params": []
  },
  "withdrawal-panel.value.fixed.changes.offset.value.totals.include.all.pages.6666c8": {
    "text": "{v0} fixed changes · Offset {v1}. Totals include all pages.",
    "params": [
      "v0",
      "v1"
    ]
  },
  "withdrawal-panel.previous.impact.changes.ab7233": {
    "text": "Previous impact changes",
    "params": []
  },
  "withdrawal-panel.next.impact.changes.15deab": {
    "text": "Next impact changes",
    "params": []
  },
  "withdrawal-panel.withdraw.permanently.07fd73": {
    "text": "Withdraw permanently",
    "params": []
  },
  "withdrawal-panel.back.to.trusted.question.snapshots.d7fd67": {
    "text": "Back to trusted question snapshots",
    "params": []
  },
  "blueprint-fields.value.five.distinct.questions.ac02dd": {
    "text": "{v0} · five distinct questions",
    "params": [
      "v0"
    ]
  },
  "blueprint-fields.instance.sources.identify.fixed.questions.generated.questions.ent.d26e6b": {
    "text": "Instance sources identify fixed questions. Generated questions enter through their exact template version.",
    "params": []
  },
  "blueprint-fields.rule.version.1.5.questions.4.correct.answers.required.current.fiv.b49274": {
    "text": "Rule version 1 · 5 questions · 4 correct answers required. Current five-question coverage is checked by Go.",
    "params": []
  },
  "command-controls.stop.waiting.d4af24": {
    "text": "Stop waiting",
    "params": []
  },
  "command-controls.the.result.is.unconfirmed.stopping.or.timing.out.does.not.prove.t.fba207": {
    "text": "The result is unconfirmed. Stopping or timing out does not prove that the server rejected the change.",
    "params": []
  },
  "command-controls.the.previous.command.is.preserved.03b9de": {
    "text": "The previous command is preserved.",
    "params": []
  },
  "command-controls.retry.sends.the.same.input.and.request.key.no.write.is.retried.au.4ced2e": {
    "text": "Retry sends the same input and request key. No write is retried automatically.",
    "params": []
  },
  "command-controls.try.again.after.value.seconds.a723d0": {
    "text": "Try again after {v0} seconds.",
    "params": [
      "v0"
    ]
  },
  "command-controls.discard.local.pending.request.14700a": {
    "text": "Discard local pending request",
    "params": []
  },
  "command-controls.verification.lasts.five.minutes.the.preserved.command.is.sent.onl.64a5cc": {
    "text": "Verification lasts five minutes. The preserved command is sent only when you choose Retry.",
    "params": []
  },
  "generation-panel.saved.revision.value.generation.check.22a232": {
    "text": "Saved revision {v0} · Generation check",
    "params": [
      "v0"
    ]
  },
  "generation-panel.machine.checks.passed.reviewer.approval.is.required.deef63": {
    "text": "Machine checks passed. Reviewer approval is required.",
    "params": []
  },
  "generation-panel.questions.need.work.before.submission.d2b52f": {
    "text": "Questions need work before submission.",
    "params": []
  },
  "generation-panel.showing.the.first.100.issues.totals.include.every.issue.b82545": {
    "text": "Showing the first 100 issues. Totals include every issue.",
    "params": []
  },
  "generation-panel.question.package.bytes.value.frozen.bytes.value.42a43d": {
    "text": "Question package bytes: {v0} · Frozen bytes: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "generation-panel.raw.combinations.value.excluded.value.independently.verified.inst.58393d": {
    "text": "Raw combinations: {v0} · Excluded: {v1} · Independently verified instances: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "generation-panel.value.value.excluded.combinations.671ebe": {
    "text": "{v0}: {v1} excluded combinations",
    "params": [
      "v0",
      "v1"
    ]
  },
  "generation-panel.effective.instances.value.fixed.value.generated.value.a62299": {
    "text": "Effective instances: {v0} · Fixed: {v1} · Generated: {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "generation-panel.core.objective.indices.value.value.3f0422": {
    "text": "Core objective indices: {v0} · {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "generation-panel.supplementary.objective.indices.value.d292f7": {
    "text": "Supplementary objective indices: {v0}",
    "params": [
      "v0"
    ]
  },
  "generation-panel.generated.previews.become.trusted.publication.content.only.after..03fe9c": {
    "text": "Generated previews become trusted publication content only after reviewer approval and activation.",
    "params": []
  },
  "generation-panel.no.sources.declared.source.verification.is.required.f5603d": {
    "text": "No sources declared. Source verification is required.",
    "params": []
  },
  "generation-panel.version.value.sha.value.777514": {
    "text": "Version {v0} · SHA {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "generation-panel.primary.knowledge.value.vvalue.60e3e9": {
    "text": "Primary knowledge: {v0} v{v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "generation-panel.verified.answer.value.6f716c": {
    "text": "Verified answer: {v0}",
    "params": [
      "v0"
    ]
  },
  "generation-panel.fixed.illustrations.fd3664": {
    "text": "Fixed illustrations",
    "params": []
  },
  "generation-panel.availability.follows.the.current.public.publication.check.attribu.c068b3": {
    "text": "Availability follows the current public publication. Check attribution in the original source.",
    "params": []
  },
  "generation-panel.independent.verification.witness.and.fixed.coverage.8269b5": {
    "text": "Independent verification witness and fixed coverage",
    "params": []
  },
  "review-panel.frozen.question.review.f18dbf": {
    "text": "FROZEN QUESTION REVIEW",
    "params": []
  },
  "review-panel.generator.versions.3bf6e1": {
    "text": "Generator versions: ",
    "params": []
  },
  "review-panel.verifier.versions.f87830": {
    "text": " · Verifier versions: ",
    "params": []
  },
  "review-panel.frozen.learning.objectives.14da3e": {
    "text": "Frozen learning objectives",
    "params": []
  },
  "review-panel.index.f48749": {
    "text": " · Index ",
    "params": []
  },
  "review-panel.frozen.source.mapping.1dd892": {
    "text": "Frozen source mapping",
    "params": []
  },
  "review-panel.batch.sha.value.source.sha.value.8530e4": {
    "text": "Batch SHA {v0} · Source SHA {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "review-panel.no.source.mapping.declared.verify.original.provenance.independent.bb78ac": {
    "text": "No source mapping declared. Verify original provenance independently.",
    "params": []
  },
  "review-panel.exact.parameters.value.f63e08": {
    "text": "Exact parameters: {v0}",
    "params": [
      "v0"
    ]
  },
  "review-panel.constraints.value.523e1b": {
    "text": "Constraints: {v0}",
    "params": [
      "v0"
    ]
  },
  "review-panel.distractors.value.403375": {
    "text": "Distractors: {v0}",
    "params": [
      "v0"
    ]
  },
  "review-panel.complete.frozen.template.and.engine.specification.5bf1c8": {
    "text": "Complete frozen template and engine specification",
    "params": []
  },
  "review-panel.complete.frozen.package.blueprints.and.reference.identities.519804": {
    "text": "Complete frozen package, blueprints and reference identities",
    "params": []
  },
  "review-panel.bound.question.instances.f83db5": {
    "text": "Bound question instances",
    "params": []
  },
  "review-panel.showing.value.of.value.instances.offset.value.every.instance.is.a.39637c": {
    "text": "Showing {v0} of {v1} instances · Offset {v2}. Every instance is available through these pages.",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "review-panel.previous.instances.e6cf69": {
    "text": "Previous instances",
    "params": []
  },
  "review-panel.next.instances.98b8ca": {
    "text": "Next instances",
    "params": []
  },
  "review-panel.authors.cannot.review.their.own.or.inherited.questions.c8a07b": {
    "text": "Authors cannot review their own or inherited questions.",
    "params": []
  },
  "review-panel.complete.all.six.checks.before.approval.8b89bd": {
    "text": "Complete all six checks before approval",
    "params": []
  },
  "review-panel.generation.review.statement.434467": {
    "text": "Generation review statement",
    "params": []
  },
  "review-panel.when.no.templates.are.present.explain.explicitly.why.generation.d.691385": {
    "text": "When no templates are present, explain explicitly why generation does not apply.",
    "params": []
  },
  "review-panel.resume.returned.workspace.716887": {
    "text": "Resume returned workspace",
    "params": []
  },
  "review-panel.back.to.question.submissions.c8d01f": {
    "text": "Back to question submissions",
    "params": []
  },
  "fixed-fields.use.a.mathematical.verification.witness.750a96": {
    "text": " Use a mathematical verification witness",
    "params": []
  },
  "fixed-fields.independent.verification.witness.078cd5": {
    "text": "Independent verification witness",
    "params": []
  },
  "fixed-fields.conceptual.choices.require.an.independent.human.mathematics.revie.61ec90": {
    "text": "Conceptual choices require an independent human mathematics review. Numeric questions require a valid witness.",
    "params": []
  },
  "draft-editor.editable.question.data.imported.save.and.validate.this.revision.b.a2437c": {
    "text": "Editable question data imported. Save and validate this revision before submitting.",
    "params": []
  },
  "draft-editor.invalid.or.oversized.editable.question.json.the.current.input.was.899b5c": {
    "text": "Invalid or oversized editable question JSON. The current input was preserved.",
    "params": []
  },
  "draft-editor.template.value.7d9545": {
    "text": "Template {v0}",
    "params": [
      "v0"
    ]
  },
  "draft-editor.fixed.question.value.944706": {
    "text": "Fixed question {v0}",
    "params": [
      "v0"
    ]
  },
  "draft-editor.blueprint.value.ee770e": {
    "text": "Blueprint {v0}",
    "params": [
      "v0"
    ]
  },
  "draft-editor.question.json.exceeds.the.request.size.limit.e70c01": {
    "text": "Question JSON exceeds the request size limit.",
    "params": []
  },
  "draft-editor.draft.saved.validate.the.new.saved.revision.3180f2": {
    "text": "Draft saved. Validate the new saved revision.",
    "params": []
  },
  "draft-editor.the.saved.revision.was.frozen.for.independent.review.23df27": {
    "text": "The saved revision was frozen for independent review.",
    "params": []
  },
  "draft-editor.saved.version.reloaded.your.editable.input.was.preserved.c167eb": {
    "text": "Saved version reloaded. Your editable input was preserved.",
    "params": []
  },
  "template-fields.value.source.value.value.83babd": {
    "text": "{v0} source {v1} {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "template-fields.value.objective.index.b0b812": {
    "text": "{v0} objective index",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.coverage.3b6324": {
    "text": "{v0} coverage",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.coverage.value.knowledge.6e063b": {
    "text": "{v0} coverage {v1} knowledge",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.unit.f2dadb": {
    "text": "{v0} unit",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.unit.value.7da153": {
    "text": "{v0} unit {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.fixed.illustration.022155": {
    "text": "{v0} fixed illustration",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.illustration.value.id.ba47fd": {
    "text": "{v0} illustration {v1} ID",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.illustration.value.sha.a322ab": {
    "text": "{v0} illustration {v1} SHA",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.primary.knowledge.99f3ec": {
    "text": "{v0} primary knowledge",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.family.ea310d": {
    "text": "{v0} family",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.operation.e1cc41": {
    "text": "{v0} operation",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.unknown.side.48d786": {
    "text": "{v0} unknown side",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.answer.type.e7124b": {
    "text": "{v0} answer type",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.numeric.format.fa8de6": {
    "text": "{v0} numeric format",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.prompt.31f4c5": {
    "text": "{v0} prompt",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.explanation.cedb6c": {
    "text": "{v0} explanation",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.parameter.b27ed8": {
    "text": "{v0} parameter",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.parameter.value.name.93d5c0": {
    "text": "{v0} parameter {v1} name",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.parameter.value.values.1a245e": {
    "text": "{v0} parameter {v1} values",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.constraint.9fe5b9": {
    "text": "{v0} constraint",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.constraint.value.27d9e1": {
    "text": "{v0} constraint {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "template-fields.value.distractor.46ec27": {
    "text": "{v0} distractor",
    "params": [
      "v0"
    ]
  },
  "template-fields.value.distractor.value.acc9f0": {
    "text": "{v0} distractor {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "publication-panel.snapshot.prepared.inspect.its.complete.fixed.difference.and.evide.4818f7": {
    "text": "Snapshot prepared. Inspect its complete fixed difference and evidence before activation.",
    "params": []
  },
  "publication-panel.snapshot.activated.both.current.heads.refreshed.550507": {
    "text": "Snapshot activated. Both current heads refreshed.",
    "params": []
  },
  "blueprint-fields.value.core.97243e": {
    "text": "{v0} core",
    "params": [
      "v0"
    ]
  },
  "blueprint-fields.value.source.value.kind.276627": {
    "text": "{v0} source {v1} kind",
    "params": [
      "v0",
      "v1"
    ]
  },
  "blueprint-fields.value.coverage.explanation.e9faf5": {
    "text": "{v0} coverage explanation",
    "params": [
      "v0"
    ]
  },
  "command-controls.password.verified.retry.the.preserved.command.when.ready.aef9f8": {
    "text": "Password verified. Retry the preserved command when ready.",
    "params": []
  },
  "review-panel.instance.data.does.not.match.this.frozen.submission.a1ce90": {
    "text": "Instance data does not match this frozen submission.",
    "params": []
  },
  "review-panel.submission.returned.for.changes.690b53": {
    "text": "Submission returned for changes.",
    "params": []
  },
  "fixed-fields.value.answer.numerator.b7eb62": {
    "text": "{v0} answer numerator",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.answer.denominator.589df2": {
    "text": "{v0} answer denominator",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.choice.a4beb3": {
    "text": "{v0} choice",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.choice.value.id.8e997c": {
    "text": "{v0} choice {v1} ID",
    "params": [
      "v0",
      "v1"
    ]
  },
  "fixed-fields.value.choice.value.text.108530": {
    "text": "{v0} choice {v1} text",
    "params": [
      "v0",
      "v1"
    ]
  },
  "fixed-fields.value.correct.choice.id.77215b": {
    "text": "{v0} correct choice ID",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.witness.family.aa4e0d": {
    "text": "{v0} witness family",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.witness.operation.4515de": {
    "text": "{v0} witness operation",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.witness.unknown.side.b51c41": {
    "text": "{v0} witness unknown side",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.witness.parameter.80503f": {
    "text": "{v0} witness parameter",
    "params": [
      "v0"
    ]
  },
  "fixed-fields.value.witness.parameter.value.name.d93d1a": {
    "text": "{v0} witness parameter {v1} name",
    "params": [
      "v0",
      "v1"
    ]
  },
  "fixed-fields.value.witness.parameter.value.exact.value.2e0a8a": {
    "text": "{v0} witness parameter {v1} exact value",
    "params": [
      "v0",
      "v1"
    ]
  },
  "question.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "question.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "question.error.AUTHENTICATION_REQUIRED": {
    "text": "Please sign in to continue.",
    "params": []
  },
  "question.error.CSRF_FAILED": {
    "text": "Request verification failed.",
    "params": []
  },
  "question.error.FORBIDDEN": {
    "text": "You do not have permission.",
    "params": []
  },
  "question.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password to continue.",
    "params": []
  },
  "question.error.NOT_FOUND": {
    "text": "Resource not found.",
    "params": []
  },
  "question.error.METHOD_NOT_ALLOWED": {
    "text": "Method not allowed.",
    "params": []
  },
  "question.error.QUESTION_DRAFT_CONFLICT": {
    "text": "This draft has changed. Reload before continuing.",
    "params": []
  },
  "question.error.QUESTION_PUBLICATION_STALE": {
    "text": "Published snapshots changed. Prepare again.",
    "params": []
  },
  "question.error.REVIEW_CONFLICT": {
    "text": "This submission already has a final decision.",
    "params": []
  },
  "question.error.IDEMPOTENCY_CONFLICT": {
    "text": "This request key was used for different input.",
    "params": []
  },
  "question.error.IMMUTABLE_CONFLICT": {
    "text": "This fixed version conflicts with saved questions.",
    "params": []
  },
  "question.error.VERSION_CONFLICT": {
    "text": "Create a new version for changed questions.",
    "params": []
  },
  "question.error.PAYLOAD_TOO_LARGE": {
    "text": "This request exceeds the size limit.",
    "params": []
  },
  "question.error.QUESTION_INVALID": {
    "text": "Question validation failed.",
    "params": []
  },
  "question.error.QUESTION_NOT_READY": {
    "text": "Complete the required questions before submitting.",
    "params": []
  },
  "question.error.QUESTION_LIMIT_EXCEEDED": {
    "text": "Split this question bank into smaller reviewed batches.",
    "params": []
  },
  "question.error.REVIEW_REQUIRED": {
    "text": "Independent review is required.",
    "params": []
  },
  "question.error.REAUTHENTICATION_REQUIRED": {
    "text": "Verify your password before continuing.",
    "params": []
  },
  "question.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "question.error.QUESTION_BANK_NOT_CONFIGURED": {
    "text": "Question management is temporarily unavailable.",
    "params": []
  },
  "question.error.AUTH_NOT_CONFIGURED": {
    "text": "Accounts are temporarily unavailable.",
    "params": []
  },
  "question.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "diff-panel.snapshot.data.does.not.match.the.selected.manifest.707687": {
    "text": "Snapshot data does not match the selected manifest.",
    "params": []
  },
  "template-fields.this.exact.published.knowledge.version.is.unavailable.validation..44e7ec": {
    "text": "This exact published knowledge version is unavailable. Validation will check the reference.",
    "params": []
  },
  "withdrawal-panel.preview.changed.read.a.fresh.impact.before.withdrawing.25242a": {
    "text": "Preview changed. Read a fresh impact before withdrawing.",
    "params": []
  },
  "withdrawal-panel.exact.version.withdrawn.permanently.refresh.published.coverage.to.fc0086": {
    "text": "Exact version withdrawn permanently. Refresh published coverage to see remaining usable pools.",
    "params": []
  },
  "question.check.mathematics": {
    "text": "Mathematics",
    "params": []
  },
  "question.check.explanations": {
    "text": "Explanations",
    "params": []
  },
  "question.check.coverage": {
    "text": "Coverage",
    "params": []
  },
  "question.check.sources": {
    "text": "Sources",
    "params": []
  },
  "question.check.illustrations": {
    "text": "Illustrations",
    "params": []
  },
  "question.check.generation": {
    "text": "Generation",
    "params": []
  },
  "question.enum.numeric": {
    "text": "numeric",
    "params": []
  },
  "question.enum.single_choice": {
    "text": "single_choice",
    "params": []
  },
  "question.enum.rational": {
    "text": "rational",
    "params": []
  },
  "question.enum.percentage": {
    "text": "percentage",
    "params": []
  },
  "question.enum.rational_arithmetic": {
    "text": "rational_arithmetic",
    "params": []
  },
  "question.enum.rational_comparison": {
    "text": "rational_comparison",
    "params": []
  },
  "question.enum.missing_operand": {
    "text": "missing_operand",
    "params": []
  },
  "question.enum.add": {
    "text": "add",
    "params": []
  },
  "question.enum.subtract": {
    "text": "subtract",
    "params": []
  },
  "question.enum.multiply": {
    "text": "multiply",
    "params": []
  },
  "question.enum.divide": {
    "text": "divide",
    "params": []
  },
  "question.enum.compare": {
    "text": "compare",
    "params": []
  },
  "question.enum.left": {
    "text": "left",
    "params": []
  },
  "question.enum.right": {
    "text": "right",
    "params": []
  },
  "question.enum.nonzero_divisor": {
    "text": "nonzero_divisor",
    "params": []
  },
  "question.enum.nonnegative_result": {
    "text": "nonnegative_result",
    "params": []
  },
  "question.enum.distinct_operands": {
    "text": "distinct_operands",
    "params": []
  },
  "question.enum.negate": {
    "text": "negate",
    "params": []
  },
  "question.enum.plus_one": {
    "text": "plus_one",
    "params": []
  },
  "question.enum.minus_one": {
    "text": "minus_one",
    "params": []
  },
  "question.enum.reciprocal": {
    "text": "reciprocal",
    "params": []
  },
  "question.enum.core": {
    "text": "core",
    "params": []
  },
  "question.enum.supplementary": {
    "text": "supplementary",
    "params": []
  },
  "question.enum.template": {
    "text": "template",
    "params": []
  },
  "question.enum.instance": {
    "text": "instance",
    "params": []
  },
  "question.enum.blueprint": {
    "text": "blueprint",
    "params": []
  },
  "publication-panel.current.snapshot.identity.does.not.match.f378f9": {
    "text": "Current snapshot identity does not match.",
    "params": []
  },
  "review-panel.administrator.self.review.saved.ea6679": {
    "text": "Administrator self-review saved.",
    "params": []
  },
  "review-panel.independent.review.saved.ba7d40": {
    "text": "Independent review saved.",
    "params": []
  },
  "question.check.objectives": {
    "text": "Objectives",
    "params": []
  },
  "history-list.no.personal.learning.records.yet.b85eae": {
    "text": "No personal learning records yet.",
    "params": []
  },
  "history-list.version.value.value.value.ea7ec3": {
    "text": "Version {v0} · {v1} · {v2}",
    "params": [
      "v0",
      "v1",
      "v2"
    ]
  },
  "history-list.value.utc.c16e72": {
    "text": "{v0} UTC",
    "params": [
      "v0"
    ]
  },
  "history-list.view.attempt.b17a6b": {
    "text": "View attempt",
    "params": []
  },
  "history-list.view.learning.record.95f5a9": {
    "text": "View learning record",
    "params": []
  },
  "history-list.history.pages.714bf6": {
    "text": "History pages",
    "params": []
  },
  "attempt-asset.question.illustration.b088b7": {
    "text": "Question illustration",
    "params": []
  },
  "path-progress.fixed.route.progress.367920": {
    "text": "Fixed route progress",
    "params": []
  },
  "path-progress.saved.route.version.value.555a99": {
    "text": "Saved route · Version {v0}",
    "params": [
      "v0"
    ]
  },
  "path-progress.value.value.read.8ae4d9": {
    "text": "{v0} / {v1} read",
    "params": [
      "v0",
      "v1"
    ]
  },
  "path-progress.value.value.passed.bb2e62": {
    "text": "{v0} / {v1} passed",
    "params": [
      "v0",
      "v1"
    ]
  },
  "path-progress.value.value.historically.unlocked.abdc56": {
    "text": "{v0} / {v1} historically unlocked",
    "params": [
      "v0",
      "v1"
    ]
  },
  "path-progress.new.route.version.available.67ae0e": {
    "text": "New route version available",
    "params": []
  },
  "path-progress.your.saved.version.is.unchanged.b5be92": {
    "text": "Your saved version is unchanged. ",
    "params": []
  },
  "path-progress.explore.the.current.route.eec21f": {
    "text": "Explore the current route",
    "params": []
  },
  "path-progress.saved.evidence.needs.a.current.review.8dbbf3": {
    "text": "Saved evidence needs a current review.",
    "params": []
  },
  "path-progress.view.saved.learning.record.3badde": {
    "text": "View saved learning record",
    "params": []
  },
  "path-progress.join.learning.route.d8d3be": {
    "text": "Join learning route",
    "params": []
  },
  "path-progress.keep.your.route.progress.a80a0c": {
    "text": "Keep your route progress",
    "params": []
  },
  "path-progress.save.this.route.version.with.value.knowledge.points.reading.it.do.96b512": {
    "text": "Save this route version with {v0} knowledge points. Reading it does not join it.",
    "params": [
      "v0"
    ]
  },
  "path-progress.join.route.5560fe": {
    "text": "Join route",
    "params": []
  },
  "overview-panel.learning.overview.f9c4ad": {
    "text": "Learning overview",
    "params": []
  },
  "overview-panel.your.progress.433a72": {
    "text": "Your progress",
    "params": []
  },
  "overview-panel.value.read.0fdd28": {
    "text": "{v0} read",
    "params": [
      "v0"
    ]
  },
  "overview-panel.value.passed.70be87": {
    "text": "{v0} passed",
    "params": [
      "v0"
    ]
  },
  "overview-panel.value.historically.unlocked.16f06c": {
    "text": "{v0} historically unlocked",
    "params": [
      "v0"
    ]
  },
  "overview-panel.value.knowledge.points.started.79dd20": {
    "text": "{v0} knowledge points started",
    "params": [
      "v0"
    ]
  },
  "overview-panel.reviewed.learning.content.will.appear.as.it.becomes.available.fdce83": {
    "text": "Reviewed learning content will appear as it becomes available.",
    "params": []
  },
  "overview-panel.continue.value.5d2b50": {
    "text": "Continue {v0}",
    "params": [
      "v0"
    ]
  },
  "overview-panel.view.all.learning.history.7a7600": {
    "text": "View all learning history",
    "params": []
  },
  "overview-panel.learning.routes.13f0bd": {
    "text": "Learning routes",
    "params": []
  },
  "overview-panel.knowledge.points.4c5586": {
    "text": " knowledge points",
    "params": []
  },
  "overview-panel.no.reviewed.routes.are.available.yet.f42dab": {
    "text": "No reviewed routes are available yet.",
    "params": []
  },
  "knowledge-controls.personal.knowledge.record.ac5aa8": {
    "text": "Personal knowledge record",
    "params": []
  },
  "knowledge-controls.historical.learning.record.1cce1f": {
    "text": "Historical learning record",
    "params": []
  },
  "knowledge-controls.your.learning.record.0e3a28": {
    "text": "Your learning record",
    "params": []
  },
  "knowledge-controls.this.is.a.saved.version.8b4989": {
    "text": "This is a saved version. ",
    "params": []
  },
  "knowledge-controls.review.the.current.explanation.8c3a53": {
    "text": "Review the current explanation",
    "params": []
  },
  "knowledge-controls.read.freely.your.learning.record.changes.when.you.choose.an.actio.8d30c1": {
    "text": "Read freely. Your learning record changes when you choose an action below.",
    "params": []
  },
  "knowledge-controls.start.learning.ac1796": {
    "text": "Start learning",
    "params": []
  },
  "knowledge-controls.mark.as.learned.0636c7": {
    "text": "Mark as learned",
    "params": []
  },
  "knowledge-controls.reading.completion.recorded.f65e0d": {
    "text": "Reading completion recorded.",
    "params": []
  },
  "knowledge-controls.work.through.the.prerequisites.or.use.a.diagnostic.assessment.to..f48ccf": {
    "text": "Work through the prerequisites, or use a diagnostic assessment to check your understanding.",
    "params": []
  },
  "knowledge-controls.ready.5fa7aa": {
    "text": "Ready",
    "params": []
  },
  "knowledge-controls.needs.assessment.ca948f": {
    "text": "Needs assessment",
    "params": []
  },
  "knowledge-controls.view.learning.history.31b66d": {
    "text": "View learning history",
    "params": []
  },
  "knowledge-controls.practice.and.assessment.24449e": {
    "text": "Practice and assessment",
    "params": []
  },
  "knowledge-controls.build.understanding.d3f7f9": {
    "text": "Build understanding",
    "params": []
  },
  "knowledge-controls.start.single.question.practice.5312a9": {
    "text": "Start single-question practice",
    "params": []
  },
  "knowledge-controls.check.your.understanding.b15e3d": {
    "text": "Check your understanding",
    "params": []
  },
  "knowledge-controls.continue.current.assessment.38cce9": {
    "text": "Continue current assessment",
    "params": []
  },
  "knowledge-controls.a.reviewed.five.question.assessment.is.not.available.for.this.kno.efc816": {
    "text": "A reviewed five-question assessment is not available for this knowledge point yet. You can still practise.",
    "params": []
  },
  "knowledge-controls.assessment.goals.bc235e": {
    "text": "Assessment goals",
    "params": []
  },
  "knowledge-controls.set.value.value.31aca9": {
    "text": "Set {v0}: {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "knowledge-controls.assessment.mode.87c89f": {
    "text": "Assessment mode",
    "params": []
  },
  "knowledge-controls.learning.check.3fa22f": {
    "text": "Learning check",
    "params": []
  },
  "knowledge-controls.diagnostic.assessment.856c34": {
    "text": "Diagnostic assessment",
    "params": []
  },
  "knowledge-controls.review.assessment.77cf32": {
    "text": "Review assessment",
    "params": []
  },
  "knowledge-controls.five.eligible.questions.are.not.available.92f917": {
    "text": "Five eligible questions are not available. ",
    "params": []
  },
  "knowledge-controls.saved.grading.evidence.is.awaiting.independent.correction.review.464d20": {
    "text": "Saved grading evidence is awaiting independent correction review.",
    "params": []
  },
  "knowledge-controls.previously.viewed.answers.need.time.before.they.can.be.used.in.an.5f58dc": {
    "text": "Previously viewed answers need time before they can be used in an assessment.",
    "params": []
  },
  "knowledge-controls.additional.reviewed.questions.are.needed.for.these.goals.b62d13": {
    "text": "Additional reviewed questions are needed for these goals.",
    "params": []
  },
  "knowledge-controls.try.after.95e48e": {
    "text": " Try after ",
    "params": []
  },
  "knowledge-controls.start.five.question.assessment.09a806": {
    "text": "Start five-question assessment",
    "params": []
  },
  "learning-status.unavailable.ca1844": {
    "text": "Unavailable",
    "params": []
  },
  "learning-status.unlocked.531c7b": {
    "text": "Unlocked",
    "params": []
  },
  "learning-status.locked.a424e3": {
    "text": "Locked",
    "params": []
  },
  "learning-status.previously.unlocked.ecd216": {
    "text": "Previously unlocked",
    "params": []
  },
  "learning-status.sign.in.to.keep.your.learning.records.c763b1": {
    "text": "Sign in to keep your learning records",
    "params": []
  },
  "learning-status.learning.records.are.not.available.yet.d0dd91": {
    "text": "Learning records are not available yet",
    "params": []
  },
  "learning-status.learning.is.temporarily.unavailable.5d2180": {
    "text": "Learning is temporarily unavailable",
    "params": []
  },
  "learning-status.checking.your.learning.account.3df2f8": {
    "text": "Checking your learning account…",
    "params": []
  },
  "learning-status.saving.23e392": {
    "text": "Saving…",
    "params": []
  },
  "learning-status.no.confirmation.received.you.can.retry.the.same.request.e68eb0": {
    "text": "No confirmation received. You can retry the same request.",
    "params": []
  },
  "learning-status.use.the.format.shown.beside.the.question.949e24": {
    "text": "Use the format shown beside the question.",
    "params": []
  },
  "learning-status.try.after.value.ea24ae": {
    "text": "Try after {v0}",
    "params": [
      "v0"
    ]
  },
  "learning-status.continue.current.attempt.9545db": {
    "text": "Continue current attempt",
    "params": []
  },
  "learning-status.retry.same.request.16003a": {
    "text": "Retry same request",
    "params": []
  },
  "knowledge-list.knowledge.learning.states.57e222": {
    "text": "Knowledge learning states",
    "params": []
  },
  "knowledge-list.your.knowledge.map.cbbacd": {
    "text": "Your knowledge map",
    "params": []
  },
  "knowledge-list.review.learning.record.de4302": {
    "text": "Review learning record",
    "params": []
  },
  "knowledge-list.no.reviewed.knowledge.points.are.available.yet.3cee4b": {
    "text": "No reviewed knowledge points are available yet.",
    "params": []
  },
  "knowledge-list.knowledge.pages.10ab02": {
    "text": "Knowledge pages",
    "params": []
  },
  "knowledge-list.previous.knowledge.points.4419a5": {
    "text": "Previous knowledge points",
    "params": []
  },
  "knowledge-list.next.knowledge.points.da8759": {
    "text": "Next knowledge points",
    "params": []
  },
  "knowledge-list.explore.your.full.learning.map.d9ec05": {
    "text": "Explore your full learning map",
    "params": []
  },
  "practice-panel.single.question.practice.7f9c5a": {
    "text": "Single-question practice",
    "params": []
  },
  "practice-panel.practice.supports.understanding.it.does.not.grant.assessment.qual.9691a0": {
    "text": "Practice supports understanding. It does not grant assessment qualification.",
    "params": []
  },
  "practice-panel.expires.at.2e4e2e": {
    "text": "Expires at ",
    "params": []
  },
  "practice-panel.submit.answer.b896ad": {
    "text": "Submit answer",
    "params": []
  },
  "practice-panel.reveal.answer.848e09": {
    "text": "Reveal answer",
    "params": []
  },
  "practice-panel.abandon.practice.8b007f": {
    "text": "Abandon practice",
    "params": []
  },
  "practice-panel.reveal.answer.ends.this.practice.immediately.c91381": {
    "text": "Reveal answer ends this practice immediately.",
    "params": []
  },
  "practice-panel.practice.ended.answer.revealed.3e12d9": {
    "text": "Practice ended: answer revealed.",
    "params": []
  },
  "practice-panel.review.the.lesson.or.start.another.practice.70e26d": {
    "text": "Review the lesson or start another practice",
    "params": []
  },
  "result-panel.question.value.016509": {
    "text": "Question {v0}",
    "params": [
      "v0"
    ]
  },
  "result-panel.answer.and.explanation.are.unavailable.because.the.saved.source.w.9cb9a8": {
    "text": "Answer and explanation are unavailable because the saved source was withdrawn.",
    "params": []
  },
  "result-panel.your.answer.value.3bb7a4": {
    "text": "Your answer: {v0}",
    "params": [
      "v0"
    ]
  },
  "result-panel.correct.aca01a": {
    "text": "Correct",
    "params": []
  },
  "result-panel.incorrect.a8a57d": {
    "text": "Incorrect",
    "params": []
  },
  "result-panel.correct.answer.valuevalue.25351e": {
    "text": "Correct answer: {v0}{v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "result-panel.correct.answer.value.5463b6": {
    "text": "Correct answer: {v0}",
    "params": [
      "v0"
    ]
  },
  "result-panel.saved.evidence.value.review.the.current.lesson.before.another.ass.c0db55": {
    "text": "Saved evidence: {v0}. Review the current lesson before another assessment.",
    "params": [
      "v0"
    ]
  },
  "result-panel.assessment.affected.1237e8": {
    "text": "Assessment affected",
    "params": []
  },
  "result-panel.this.assessment.could.not.be.graded.reliably.start.a.fresh.assess.b8d898": {
    "text": "This assessment could not be graded reliably. Start a fresh assessment when eligible questions are available.",
    "params": []
  },
  "result-panel.passed.436fe7": {
    "text": "Passed",
    "params": []
  },
  "result-panel.failed.031a8f": {
    "text": "Failed",
    "params": []
  },
  "result-panel.four.correct.answers.out.of.five.are.required.to.pass.7d4df2": {
    "text": "Four correct answers out of five are required to pass.",
    "params": []
  },
  "result-panel.saved.result.needs.review.e535eb": {
    "text": "Saved result needs review",
    "params": []
  },
  "result-panel.the.original.score.is.preserved.this.saved.evidence.may.no.longer.a42a52": {
    "text": "The original score is preserved. This saved evidence may no longer qualify for current learning progress.",
    "params": []
  },
  "result-panel.historical.unlocks.remain.in.your.learning.record.including.after.ba32e5": {
    "text": "Historical unlocks remain in your learning record, including after an unsuccessful review.",
    "params": []
  },
  "result-panel.knowledge.qualification.recorded.03cba2": {
    "text": "Knowledge qualification recorded.",
    "params": []
  },
  "result-panel.unlocked.when.submitted.d7ae9a": {
    "text": " · Unlocked when submitted",
    "params": []
  },
  "result-panel.review.the.current.lesson.85bd8a": {
    "text": "Review the current lesson",
    "params": []
  },
  "result-panel.return.to.my.learning.a18bde": {
    "text": "Return to my learning",
    "params": []
  },
  "answer-fields.answer.for.question.value.273662": {
    "text": "Answer for question {v0}",
    "params": [
      "v0"
    ]
  },
  "answer-fields.value.at.most.128.characters.your.spelling.is.preserved.4b1382": {
    "text": "{v0} At most 128 characters. Your spelling is preserved.",
    "params": [
      "v0"
    ]
  },
  "assessment-panel.five.question.assessment.96921d": {
    "text": "Five-question assessment",
    "params": []
  },
  "assessment-panel.this.assessment.is.value.abee64": {
    "text": "This assessment is {v0}.",
    "params": [
      "v0"
    ]
  },
  "assessment-panel.view.result.fdb7ea": {
    "text": "View result",
    "params": []
  },
  "assessment-panel.return.to.the.lesson.e2abbd": {
    "text": "Return to the lesson",
    "params": []
  },
  "assessment-panel.submit.all.five.answers.together.select.skip.for.any.question.you.0824a1": {
    "text": "Submit all five answers together. Select Skip for any question you leave unanswered. Refreshing clears unsent answers; the attempt remains available.",
    "params": []
  },
  "assessment-panel.skip.question.946648": {
    "text": "Skip question ",
    "params": []
  },
  "assessment-panel.submit.five.answers.e8b79c": {
    "text": "Submit five answers",
    "params": []
  },
  "assessment-panel.abandon.assessment.d4090a": {
    "text": "Abandon assessment",
    "params": []
  },
  "assessment-panel.result.not.confirmed.a5d341": {
    "text": "Result not confirmed",
    "params": []
  },
  "learning.error.LEARNING_NOT_CONFIGURED": {
    "text": "Learning is temporarily unavailable.",
    "params": []
  },
  "learning.error.LEARNING_VERSION_STALE": {
    "text": "Published learning content changed. Reload before continuing.",
    "params": []
  },
  "learning.error.LEARNING_PREREQUISITES_UNMET": {
    "text": "Complete the prerequisites or take a diagnostic assessment.",
    "params": []
  },
  "learning.error.ASSESSMENT_NOT_READY": {
    "text": "Five eligible questions are not available yet.",
    "params": []
  },
  "learning.error.ASSESSMENT_ACTIVE": {
    "text": "Continue or abandon the existing attempt.",
    "params": []
  },
  "learning.error.ASSESSMENT_EXPIRED": {
    "text": "This attempt has expired.",
    "params": []
  },
  "learning.error.ASSESSMENT_STATE_CONFLICT": {
    "text": "This attempt cannot accept the action.",
    "params": []
  },
  "learning.error.ANSWER_FORMAT_INVALID": {
    "text": "Check the requested answer format.",
    "params": []
  },
  "learning.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "learning.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "learning.error.INVALID_CREDENTIALS": {
    "text": "Invalid username or password.",
    "params": []
  },
  "learning.error.AUTHENTICATION_REQUIRED": {
    "text": "Please sign in to continue.",
    "params": []
  },
  "learning.error.CSRF_FAILED": {
    "text": "Request verification failed.",
    "params": []
  },
  "learning.error.FORBIDDEN": {
    "text": "You do not have permission.",
    "params": []
  },
  "learning.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password to continue.",
    "params": []
  },
  "learning.error.NOT_FOUND": {
    "text": "Resource not found.",
    "params": []
  },
  "learning.error.METHOD_NOT_ALLOWED": {
    "text": "Method not allowed.",
    "params": []
  },
  "learning.error.IDEMPOTENCY_CONFLICT": {
    "text": "This request key was used for different input.",
    "params": []
  },
  "learning.error.PAYLOAD_TOO_LARGE": {
    "text": "This request exceeds the size limit.",
    "params": []
  },
  "learning.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "learning.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "learning.error.AUTH_NOT_CONFIGURED": {
    "text": "Accounts are temporarily unavailable.",
    "params": []
  },
  "learning.error.USERNAME_UNAVAILABLE": {
    "text": "This username is unavailable.",
    "params": []
  },
  "learning.error.ALREADY_AUTHENTICATED": {
    "text": "Sign out before using another account.",
    "params": []
  },
  "learning.error.LAST_ADMIN_REQUIRED": {
    "text": "At least one administrator is required.",
    "params": []
  },
  "learning.error.REAUTHENTICATION_REQUIRED": {
    "text": "Verify your password before continuing.",
    "params": []
  },
  "learning.state.unlearned": {
    "text": "Unlearned",
    "params": []
  },
  "learning.state.learning": {
    "text": "Learning",
    "params": []
  },
  "learning.state.learned": {
    "text": "Learned",
    "params": []
  },
  "learning.state.needs-review": {
    "text": "Needs review",
    "params": []
  },
  "learning.state.mastered": {
    "text": "Mastered",
    "params": []
  },
  "learning.format.INVALID_SYNTAX": {
    "text": "INVALID_SYNTAX",
    "params": []
  },
  "learning.format.INPUT_TOO_LONG": {
    "text": "INPUT_TOO_LONG",
    "params": []
  },
  "learning.format.ZERO_DENOMINATOR": {
    "text": "ZERO_DENOMINATOR",
    "params": []
  },
  "learning.format.PERCENT_REQUIRED": {
    "text": "PERCENT_REQUIRED",
    "params": []
  },
  "learning.format.RESULT_TOO_LARGE": {
    "text": "RESULT_TOO_LARGE",
    "params": []
  },
  "learning.format.INVALID_MODE": {
    "text": "INVALID_MODE",
    "params": []
  },
  "learning.answer.percentage": {
    "text": "Use a percentage with %, such as 50%.",
    "params": []
  },
  "learning.answer.number": {
    "text": "Use an integer, decimal, or fraction, such as 2, 0.5, or 1/2.",
    "params": []
  },
  "page.invalid.learning.parameters.29cb84": {
    "text": "Invalid learning parameters.",
    "params": []
  },
  "page.reset.learning.view.061672": {
    "text": "Reset learning view",
    "params": []
  },
  "page.previous.nodes.44fb9e": {
    "text": "Previous nodes",
    "params": []
  },
  "page.next.nodes.dbf84b": {
    "text": "Next nodes",
    "params": []
  },
  "page.back.to.my.learning.e67643": {
    "text": "Back to my learning",
    "params": []
  },
  "page.your.saved.routes.57bbd0": {
    "text": "Your saved routes",
    "params": []
  },
  "page.value.saved.version.value.04f7f0": {
    "text": "{v0} · Saved version {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "page.choose.a.reviewed.route.then.select.join.route.to.save.your.progr.8cbbe5": {
    "text": "Choose a reviewed route, then select Join route to save your progress.",
    "params": []
  },
  "page.previous.routes.eb58cc": {
    "text": "Previous routes",
    "params": []
  },
  "page.next.routes.e5621d": {
    "text": "Next routes",
    "params": []
  },
  "page.one.idea.at.a.time.070016": {
    "text": "ONE IDEA AT A TIME",
    "params": []
  },
  "page.see.what.you.have.read.assessed.and.unlocked.fe8551": {
    "text": "See what you have read, assessed, and unlocked.",
    "params": []
  },
  "page.invalid.history.parameters.7f8f9a": {
    "text": "Invalid history parameters.",
    "params": []
  },
  "page.reset.history.view.bc760c": {
    "text": "Reset history view",
    "params": []
  },
  "page.your.saved.versions.and.assessment.records.85765b": {
    "text": "Your saved versions and assessment records.",
    "params": []
  },
  "learning.page.routePages": {
    "text": "Route node pages",
    "params": []
  },
  "learning.page.savedPages": {
    "text": "Saved route pages",
    "params": []
  },
  "new-form.describe.what.happened.and.what.would.help.your.original.report.s.3a0e68": {
    "text": "Describe what happened and what would help. Your original report stays in the discussion history.",
    "params": []
  },
  "new-form.category.292c06": {
    "text": "Category",
    "params": []
  },
  "new-form.title.7e8cd2": {
    "text": "Title",
    "params": []
  },
  "new-form.value.120.characters.67f5dd": {
    "text": "{v0}/120 characters",
    "params": [
      "v0"
    ]
  },
  "new-form.where.on.the.page.2e6785": {
    "text": "Where on the page?",
    "params": []
  },
  "new-form.value.400.characters.c1548a": {
    "text": "{v0}/400 characters",
    "params": [
      "v0"
    ]
  },
  "new-form.details.45989d": {
    "text": "Details",
    "params": []
  },
  "new-form.value.4000.characters.9daa9f": {
    "text": "{v0}/4000 characters",
    "params": [
      "v0"
    ]
  },
  "new-form.submit.report.b41fd5": {
    "text": "Submit report",
    "params": []
  },
  "new-form.refresh.report.target.42c32c": {
    "text": "Refresh report target",
    "params": []
  },
  "new-form.refreshing.report.target.facb17": {
    "text": "Refreshing report target…",
    "params": []
  },
  "new-form.report.target.refreshed.review.the.version.before.submitting.agai.33a9e0": {
    "text": "Report target refreshed. Review the version before submitting again.",
    "params": []
  },
  "feedback-account.checking.your.feedback.account.08c898": {
    "text": "Checking your feedback account…",
    "params": []
  },
  "status.reload.page.437d0d": {
    "text": "Reload page",
    "params": []
  },
  "status.your.change.was.saved.retry.the.same.request.to.load.the.latest.s.01d912": {
    "text": "Your change was saved. Retry the same request to load the latest status.",
    "params": []
  },
  "status.edit.as.a.new.request.356c78": {
    "text": "Edit as a new request",
    "params": []
  },
  "ticket-list.feedback.review.queue.55f05a": {
    "text": "Feedback review queue",
    "params": []
  },
  "ticket-list.status.920e41": {
    "text": "Status",
    "params": []
  },
  "ticket-list.all.statuses.8ee573": {
    "text": "All statuses",
    "params": []
  },
  "ticket-list.all.categories.9d5097": {
    "text": "All categories",
    "params": []
  },
  "ticket-list.filter.queue.f927f0": {
    "text": "Filter queue",
    "params": []
  },
  "ticket-list.new.report.16f776": {
    "text": "New report",
    "params": []
  },
  "ticket-list.event.5a4db3": {
    "text": " · Event ",
    "params": []
  },
  "ticket-list.no.reports.match.this.queue.11ebb7": {
    "text": "No reports match this queue.",
    "params": []
  },
  "ticket-list.no.reports.yet.be893e": {
    "text": "No reports yet.",
    "params": []
  },
  "ticket-list.next.reports.4eb46c": {
    "text": "Next reports",
    "params": []
  },
  "discussion-panel.discussion.5eb6cf": {
    "text": "Discussion",
    "params": []
  },
  "discussion-panel.reading.874bfa": {
    "text": "Reading…",
    "params": []
  },
  "discussion-panel.read.discussion.60768c": {
    "text": "Read discussion",
    "params": []
  },
  "discussion-panel.event.133a15": {
    "text": "Event ",
    "params": []
  },
  "discussion-panel.report.author.24263d": {
    "text": "Report author",
    "params": []
  },
  "discussion-panel.review.team.4433d3": {
    "text": "Review team",
    "params": []
  },
  "discussion-panel.basis.5690cd": {
    "text": "Basis: ",
    "params": []
  },
  "discussion-panel.withdrawal.f3cdbc": {
    "text": " · Withdrawal ",
    "params": []
  },
  "discussion-panel.replacement.aa0aec": {
    "text": " · Replacement ",
    "params": []
  },
  "discussion-panel.publication.6dbeee": {
    "text": " · Publication ",
    "params": []
  },
  "discussion-panel.related.report.db9c9f": {
    "text": " · Related report ",
    "params": []
  },
  "discussion-panel.next.discussion.page.007355": {
    "text": "Next discussion page",
    "params": []
  },
  "discussion-panel.back.to.reports.b62a50": {
    "text": "Back to reports",
    "params": []
  },
  "discussion-panel.current.basis.value.9e8023": {
    "text": "Current basis: {v0}",
    "params": [
      "v0"
    ]
  },
  "discussion-panel.reload.report.status.60577f": {
    "text": "Reload report status",
    "params": []
  },
  "discussion-panel.additional.details.00fcfc": {
    "text": "Additional details",
    "params": []
  },
  "discussion-panel.add.details.78056f": {
    "text": "Add details",
    "params": []
  },
  "review-panel.handle.this.report.3fef67": {
    "text": "Handle this report",
    "params": []
  },
  "review-panel.report.status.22a134": {
    "text": "Report status",
    "params": []
  },
  "review-panel.value.choose.an.available.status.bb7389": {
    "text": "{v0} · choose an available status",
    "params": [
      "v0"
    ]
  },
  "review-panel.review.reply.6bf2f2": {
    "text": "Review reply",
    "params": []
  },
  "review-panel.resolution.basis.47ef7d": {
    "text": "Resolution basis",
    "params": []
  },
  "review-panel.duplicate.report.id.2eed31": {
    "text": "Duplicate report ID",
    "params": []
  },
  "review-panel.withdrawal.source.77a1ed": {
    "text": "Withdrawal source",
    "params": []
  },
  "review-panel.content.47bd29": {
    "text": "Content",
    "params": []
  },
  "review-panel.question.bank.d8c022": {
    "text": "Question bank",
    "params": []
  },
  "review-panel.withdrawal.event.id.935c33": {
    "text": "Withdrawal event ID",
    "params": []
  },
  "review-panel.use.an.independently.approved.already.published.replacement.b34ac2": {
    "text": "Use an independently approved, already published replacement.",
    "params": []
  },
  "review-panel.replacement.kind.cdfd5f": {
    "text": "Replacement kind",
    "params": []
  },
  "review-panel.replacement.id.114847": {
    "text": "Replacement ID",
    "params": []
  },
  "review-panel.replacement.version.bd1b6d": {
    "text": "Replacement version",
    "params": []
  },
  "review-panel.replacement.sha256.1002ff": {
    "text": "Replacement SHA256",
    "params": []
  },
  "review-panel.replacement.publication.id.6723c8": {
    "text": "Replacement publication ID",
    "params": []
  },
  "review-panel.save.handling.result.fe5bce": {
    "text": "Save handling result",
    "params": []
  },
  "correction-account.checking.your.correction.account.da4282": {
    "text": "Checking your correction account…",
    "params": []
  },
  "result-panel.learning.correction.c9d5ff": {
    "text": "Learning correction",
    "params": []
  },
  "result-panel.score.value.5.value.68a83e": {
    "text": "Score: {v0}/5 · {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "result-panel.current.validity.value.a14551": {
    "text": "Current validity: {v0}",
    "params": [
      "v0"
    ]
  },
  "result-panel.this.correction.currently.cannot.provide.a.learning.qualification.8712ec": {
    "text": "This correction currently cannot provide a learning qualification.",
    "params": []
  },
  "result-panel.your.original.answers.are.preserved.while.the.basis.is.independen.dfaf01": {
    "text": "Your original answers are preserved while the basis is independently reviewed.",
    "params": []
  },
  "result-panel.review.the.current.knowledge.and.take.a.new.assessment.when.it.be.c48272": {
    "text": "Review the current knowledge and take a new assessment when it becomes available.",
    "params": []
  },
  "result-panel.return.to.the.updated.material.before.continuing.57ef79": {
    "text": "Return to the updated material before continuing.",
    "params": []
  },
  "result-panel.review.knowledge.4b5f8a": {
    "text": "Review knowledge",
    "params": []
  },
  "result-panel.original.result.34bd06": {
    "text": "Original result",
    "params": []
  },
  "result-panel.previous.correction.c0de64": {
    "text": "Previous correction",
    "params": []
  },
  "result-panel.reload.correction.status.e893ee": {
    "text": "Reload correction status",
    "params": []
  },
  "result-panel.review.corrected.answers.83394a": {
    "text": "Review corrected answers",
    "params": []
  },
  "result-panel.corrected.result.def77c": {
    "text": "Corrected result",
    "params": []
  },
  "result-panel.the.same.original.answers.were.checked.against.the.independently..edd1c9": {
    "text": "The same original answers were checked against the independently approved basis.",
    "params": []
  },
  "result-panel.original.answer.value.80de39": {
    "text": "Original answer: {v0}",
    "params": [
      "v0"
    ]
  },
  "result-panel.not.correct.f9cade": {
    "text": "Not correct",
    "params": []
  },
  "result-panel.original.value.vvalue.effective.value.vvalue.28ac10": {
    "text": "Original {v0} v{v1} → Effective {v2} v{v3}",
    "params": [
      "v0",
      "v1",
      "v2",
      "v3"
    ]
  },
  "qualification-link.view.corrected.qualification.742e9a": {
    "text": "View corrected qualification",
    "params": []
  },
  "qualification-link.original.assessment.cebd35": {
    "text": "Original assessment",
    "params": []
  },
  "case-list.follow.affected.learning.evidence.from.a.registered.issue.to.an.i.e9df91": {
    "text": "Follow affected learning evidence from a registered issue to an independently reviewed correction.",
    "params": []
  },
  "case-list.reload.correction.cases.01b77f": {
    "text": "Reload correction cases",
    "params": []
  },
  "case-list.grading.rule.review.ab2fe4": {
    "text": "Grading rule review",
    "params": []
  },
  "case-list.withdrawn.source.review.966bbb": {
    "text": "Withdrawn source review",
    "params": []
  },
  "case-list.approved.basis.available.c1fcbf": {
    "text": "Approved basis available",
    "params": []
  },
  "case-list.awaiting.an.approved.basis.0eaba0": {
    "text": "Awaiting an approved basis",
    "params": []
  },
  "case-list.no.correction.cases.yet.19127d": {
    "text": "No correction cases yet.",
    "params": []
  },
  "case-list.next.cases.f41e3e": {
    "text": "Next cases",
    "params": []
  },
  "case-list.register.a.correction.case.c8261a": {
    "text": "Register a correction case",
    "params": []
  },
  "case-list.a.grading.issue.uses.the.server.registration.time.withdrawn.sourc.cce17d": {
    "text": "A grading issue uses the server registration time. Withdrawn sources must reference an actual withdrawal event.",
    "params": []
  },
  "case-list.case.type.858985": {
    "text": "Case type",
    "params": []
  },
  "case-list.withdrawn.source.749b87": {
    "text": "Withdrawn source",
    "params": []
  },
  "case-list.grading.rule.2115fc": {
    "text": "Grading rule",
    "params": []
  },
  "case-list.learning.content.ace9d7": {
    "text": "Learning content",
    "params": []
  },
  "case-list.rule.version.1.exact.rational.and.single.choice.grading.80f013": {
    "text": "Rule version 1 · Exact rational and single choice grading.",
    "params": []
  },
  "case-list.grading.scope.ea1ec5": {
    "text": "Grading scope",
    "params": []
  },
  "case-list.all.mathematics.evidence.9fc01d": {
    "text": "All mathematics evidence",
    "params": []
  },
  "case-list.one.knowledge.version.0c6041": {
    "text": "One knowledge version",
    "params": []
  },
  "case-list.knowledge.id.d157a6": {
    "text": "Knowledge ID",
    "params": []
  },
  "case-list.knowledge.version.acf62a": {
    "text": "Knowledge version",
    "params": []
  },
  "case-list.knowledge.sha256.46465c": {
    "text": "Knowledge SHA256",
    "params": []
  },
  "case-list.register.correction.case.abccfa": {
    "text": "Register correction case",
    "params": []
  },
  "case-list.open.correction.case.7931bb": {
    "text": "Open correction case",
    "params": []
  },
  "case-panel.awaiting.approval.ae25c9": {
    "text": "Awaiting approval",
    "params": []
  },
  "case-panel.original.attempts.created.before.value.are.within.scope.007445": {
    "text": "Original attempts created before {v0} are within scope.",
    "params": [
      "v0"
    ]
  },
  "case-panel.reload.case.status.665567": {
    "text": "Reload case status",
    "params": []
  },
  "case-panel.correction.plans.dfd490": {
    "text": "Correction plans",
    "params": []
  },
  "case-panel.plan.version.value.value.b562a0": {
    "text": "Plan version {v0} · {v1}",
    "params": [
      "v0",
      "v1"
    ]
  },
  "case-panel.sequence.value.value.exact.instance.mappings.7c40db": {
    "text": "Sequence {v0} · {v1} exact instance mappings",
    "params": [
      "v0",
      "v1"
    ]
  },
  "case-panel.no.correction.plan.yet.91fe2a": {
    "text": "No correction plan yet.",
    "params": []
  },
  "case-panel.more.plans.4b0125": {
    "text": "More plans",
    "params": []
  },
  "case-panel.draft.a.correction.plan.af8cb8": {
    "text": "Draft a correction plan",
    "params": []
  },
  "case-panel.plan.reason.dc2c72": {
    "text": "Plan reason",
    "params": []
  },
  "case-panel.create.correction.plan.23b28c": {
    "text": "Create correction plan",
    "params": []
  },
  "case-panel.open.correction.plan.5cd0c5": {
    "text": "Open correction plan",
    "params": []
  },
  "case-panel.impact.processing.5455c8": {
    "text": "Impact processing",
    "params": []
  },
  "case-panel.processing.retries.preserve.completed.evidence.and.the.audit.hist.e975f9": {
    "text": "Processing retries preserve completed evidence and the audit history.",
    "params": []
  },
  "case-panel.value.evidence.records.checked.attempt.value.8.5baf30": {
    "text": "{v0} evidence records checked · Attempt {v1}/8",
    "params": [
      "v0",
      "v1"
    ]
  },
  "case-panel.error.category.value.e53a05": {
    "text": "Error category: {v0}",
    "params": [
      "v0"
    ]
  },
  "case-panel.next.attempt.value.cd648a": {
    "text": "Next attempt: {v0}",
    "params": [
      "v0"
    ]
  },
  "case-panel.retry.processing.593933": {
    "text": "Retry processing",
    "params": []
  },
  "case-panel.more.jobs.a1464e": {
    "text": "More jobs",
    "params": []
  },
  "evidence-link.corrections.for.saved.evidence.09b140": {
    "text": "Corrections for saved evidence",
    "params": []
  },
  "evidence-link.check.corrections.cddc7d": {
    "text": "Check corrections",
    "params": []
  },
  "evidence-link.checking.corrections.e0e9d6": {
    "text": "Checking corrections…",
    "params": []
  },
  "evidence-link.no.correction.results.yet.be7993": {
    "text": "No correction results yet.",
    "params": []
  },
  "evidence-link.view.correction.13dc93": {
    "text": "View correction",
    "params": []
  },
  "evidence-link.more.corrections.56d196": {
    "text": "More corrections",
    "params": []
  },
  "status.password.verified.retry.the.preserved.request.15ada6": {
    "text": "Password verified. Retry the preserved request.",
    "params": []
  },
  "status.checking.and.saving.ff6df1": {
    "text": "Checking and saving…",
    "params": []
  },
  "status.verification.lasts.five.minutes.retry.the.preserved.command.separ.043e8e": {
    "text": "Verification lasts five minutes. Retry the preserved command separately.",
    "params": []
  },
  "plan-editor.exact.instance.mappings.bdfd3a": {
    "text": "Exact instance mappings",
    "params": []
  },
  "plan-editor.keep.question.meaning.and.parameters.unchanged.only.an.independen.89968d": {
    "text": "Keep question meaning and parameters unchanged. Only an independently published correction to answers or explanations may replace an instance.",
    "params": []
  },
  "plan-editor.mapping.value.fc087c": {
    "text": "Mapping {v0}",
    "params": [
      "v0"
    ]
  },
  "plan-editor.original.instance.4f3d6e": {
    "text": "Original instance",
    "params": []
  },
  "plan-editor.approved.replacement.f4ed84": {
    "text": "Approved replacement",
    "params": []
  },
  "plan-editor.publication.id.8130b8": {
    "text": " publication ID",
    "params": []
  },
  "plan-editor.remove.mapping.value.5a5ea7": {
    "text": "Remove mapping {v0}",
    "params": [
      "v0"
    ]
  },
  "plan-editor.add.instance.mapping.77231b": {
    "text": "Add instance mapping",
    "params": []
  },
  "plan-editor.correction.plan.version.value.8aa1ce": {
    "text": "Correction plan · Version {v0}",
    "params": [
      "v0"
    ]
  },
  "plan-editor.sequence.value.b4d4eb": {
    "text": "Sequence {v0}",
    "params": [
      "v0"
    ]
  },
  "plan-editor.mappings.e7ca2a": {
    "text": " mappings",
    "params": []
  },
  "plan-editor.this.draft.has.not.been.sealed.4a1611": {
    "text": "This draft has not been sealed.",
    "params": []
  },
  "plan-editor.reload.plan.status.082e03": {
    "text": "Reload plan status",
    "params": []
  },
  "plan-editor.read.plan.details.5c6186": {
    "text": "Read plan details",
    "params": []
  },
  "plan-editor.new.plan.version.a46c49": {
    "text": "New plan version",
    "params": []
  },
  "plan-editor.approved.basis.and.draft.538ca7": {
    "text": "Approved basis and draft",
    "params": []
  },
  "plan-editor.review.decision.value.f55c2d": {
    "text": "Review decision: {v0}",
    "params": [
      "v0"
    ]
  },
  "plan-editor.draft.a.new.plan.version.6ef060": {
    "text": "Draft a new plan version",
    "params": []
  },
  "plan-editor.create.new.plan.version.88490f": {
    "text": "Create new plan version",
    "params": []
  },
  "plan-editor.save.plan.draft.8fd425": {
    "text": "Save plan draft",
    "params": []
  },
  "plan-editor.submit.for.independent.review.7dad26": {
    "text": "Submit for independent review",
    "params": []
  },
  "plan-editor.open.new.plan.version.4e2487": {
    "text": "Open new plan version",
    "params": []
  },
  "review-panel.approve.only.after.checking.the.exact.published.sources.creators..0f5188": {
    "text": "Approve only after checking the exact published sources. Creators, editors of this draft and authors of its mathematical sources cannot approve it.",
    "params": []
  },
  "review-panel.independent.review.reason.08bd46": {
    "text": "Independent review reason",
    "params": []
  },
  "review-panel.approve.correction.d71667": {
    "text": "Approve correction",
    "params": []
  },
  "review-panel.reject.correction.0558f4": {
    "text": "Reject correction",
    "params": []
  },
  "inbox.value.unread.notifications.39c4b8": {
    "text": "{v0} unread notifications",
    "params": [
      "v0"
    ]
  },
  "inbox.reload.notifications.7aeb54": {
    "text": "Reload notifications",
    "params": []
  },
  "inbox.read.9b9a8d": {
    "text": "Read",
    "params": []
  },
  "inbox.unread.1b9f38": {
    "text": "Unread",
    "params": []
  },
  "inbox.mark.as.read.50c8b8": {
    "text": "Mark as read",
    "params": []
  },
  "inbox.you.have.no.notifications.yet.9654ca": {
    "text": "You have no notifications yet.",
    "params": []
  },
  "inbox.next.notifications.8c2b27": {
    "text": "Next notifications",
    "params": []
  },
  "feedback.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "feedback.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "feedback.error.AUTHENTICATION_REQUIRED": {
    "text": "Sign in before continuing.",
    "params": []
  },
  "feedback.error.CSRF_FAILED": {
    "text": "Refresh your sign-in before continuing.",
    "params": []
  },
  "feedback.error.FORBIDDEN": {
    "text": "You do not have permission for this action.",
    "params": []
  },
  "feedback.error.NOT_FOUND": {
    "text": "This report or source is unavailable.",
    "params": []
  },
  "feedback.error.METHOD_NOT_ALLOWED": {
    "text": "This action is unavailable.",
    "params": []
  },
  "feedback.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password before continuing.",
    "params": []
  },
  "feedback.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "feedback.error.IDEMPOTENCY_CONFLICT": {
    "text": "This request key was used for different input.",
    "params": []
  },
  "feedback.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "feedback.error.FEEDBACK_NOT_CONFIGURED": {
    "text": "Feedback is temporarily unavailable.",
    "params": []
  },
  "feedback.error.FEEDBACK_CONFLICT": {
    "text": "This report changed. Reload before continuing.",
    "params": []
  },
  "feedback.error.FEEDBACK_TARGET_STALE": {
    "text": "This source changed. Reload before reporting.",
    "params": []
  },
  "feedback.error.FEEDBACK_ANSWER_OVERLAP": {
    "text": "Finish or leave the overlapping assessment before reading this discussion.",
    "params": []
  },
  "correction.error.INVALID_REQUEST": {
    "text": "Invalid request.",
    "params": []
  },
  "correction.error.INVALID_COOKIE": {
    "text": "Invalid sign-in cookie.",
    "params": []
  },
  "correction.error.AUTHENTICATION_REQUIRED": {
    "text": "Sign in before continuing.",
    "params": []
  },
  "correction.error.CSRF_FAILED": {
    "text": "Refresh your sign-in before continuing.",
    "params": []
  },
  "correction.error.FORBIDDEN": {
    "text": "You do not have permission for this action.",
    "params": []
  },
  "correction.error.NOT_FOUND": {
    "text": "This correction or notification is unavailable.",
    "params": []
  },
  "correction.error.METHOD_NOT_ALLOWED": {
    "text": "This action is unavailable.",
    "params": []
  },
  "correction.error.PASSWORD_CHANGE_REQUIRED": {
    "text": "Change your password before continuing.",
    "params": []
  },
  "correction.error.REAUTHENTICATION_REQUIRED": {
    "text": "Confirm your password before continuing.",
    "params": []
  },
  "correction.error.RATE_LIMITED": {
    "text": "Too many requests. Try again later.",
    "params": []
  },
  "correction.error.IDEMPOTENCY_CONFLICT": {
    "text": "This request key was used for different input.",
    "params": []
  },
  "correction.error.SERVICE_UNAVAILABLE": {
    "text": "Service temporarily unavailable.",
    "params": []
  },
  "correction.error.CORRECTION_NOT_CONFIGURED": {
    "text": "Corrections are temporarily unavailable.",
    "params": []
  },
  "correction.error.CORRECTION_CONFLICT": {
    "text": "This correction changed. Reload before continuing.",
    "params": []
  },
  "correction.error.CORRECTION_SOURCE_STALE": {
    "text": "This source changed. Reload before continuing.",
    "params": []
  },
  "correction.error.CORRECTION_ANSWER_OVERLAP": {
    "text": "Finish or leave the overlapping assessment before reading this correction.",
    "params": []
  },
  "correction.error.CORRECTION_LEASE_LOST": {
    "text": "This job lease changed. Reload before continuing.",
    "params": []
  },
  "feedback.status.new": {
    "text": "New",
    "params": []
  },
  "feedback.status.processing": {
    "text": "In progress",
    "params": []
  },
  "feedback.status.waiting_details": {
    "text": "Waiting for details",
    "params": []
  },
  "feedback.status.resolved": {
    "text": "Resolved",
    "params": []
  },
  "feedback.status.closed": {
    "text": "Closed",
    "params": []
  },
  "correction.status.corrected_passed": {
    "text": "Corrected result · Passed",
    "params": []
  },
  "correction.status.corrected_failed": {
    "text": "Corrected result · Not passed",
    "params": []
  },
  "correction.status.retake_required": {
    "text": "Retake required",
    "params": []
  },
  "correction.status.review_material": {
    "text": "Review material",
    "params": []
  },
  "correction.status.checked_unaffected": {
    "text": "Checked · Unaffected",
    "params": []
  },
  "correction.status.awaiting_review": {
    "text": "Checking · Awaiting independent review",
    "params": []
  },
  "notification.type.checking": {
    "text": "Your learning evidence is being checked.",
    "params": []
  },
  "notification.type.corrected": {
    "text": "A corrected result is ready to review.",
    "params": []
  },
  "notification.type.retake": {
    "text": "A new assessment is required.",
    "params": []
  },
  "notification.type.review_material": {
    "text": "Learning material needs another review.",
    "params": []
  },
  "notification.type.path_unavailable": {
    "text": "A learning path is currently unavailable.",
    "params": []
  },
  "feedback.category.math_error": {
    "text": "math error",
    "params": []
  },
  "feedback.category.explanation": {
    "text": "explanation",
    "params": []
  },
  "feedback.category.illustration": {
    "text": "illustration",
    "params": []
  },
  "feedback.category.reference": {
    "text": "reference",
    "params": []
  },
  "feedback.category.typo": {
    "text": "typo",
    "params": []
  },
  "feedback.category.other": {
    "text": "other",
    "params": []
  },
  "feedback.category.site": {
    "text": "site",
    "params": []
  },
  "feedback.category.answer_error": {
    "text": "answer error",
    "params": []
  },
  "feedback.category.grading_error": {
    "text": "grading error",
    "params": []
  },
  "feedback.category.technical_issue": {
    "text": "technical issue",
    "params": []
  },
  "feedback.category.accessibility": {
    "text": "accessibility",
    "params": []
  },
  "feedback.category.unclear_explanation": {
    "text": "unclear explanation",
    "params": []
  },
  "feedback.category.suggestion": {
    "text": "suggestion",
    "params": []
  },
  "feedback.basis.duplicate": {
    "text": "duplicate",
    "params": []
  },
  "feedback.basis.not_reproducible": {
    "text": "not reproducible",
    "params": []
  },
  "feedback.basis.out_of_scope": {
    "text": "out of scope",
    "params": []
  },
  "feedback.basis.suggestion_recorded": {
    "text": "suggestion recorded",
    "params": []
  },
  "feedback.basis.clarified": {
    "text": "clarified",
    "params": []
  },
  "feedback.basis.withdrawn": {
    "text": "withdrawn",
    "params": []
  },
  "feedback.basis.revision_published": {
    "text": "revision published",
    "params": []
  },
  "feedback.basis.service_fixed": {
    "text": "service fixed",
    "params": []
  },
  "knowledge-view.derivations.109f27": {
    "text": "Derivations",
    "params": []
  },
  "knowledge-view.related.ideas.e17c80": {
    "text": "Related ideas",
    "params": []
  },
  "page.you.are.signed.in.2d55cb": {
    "text": "You are signed in",
    "params": []
  },
  "audit.illustration": {
    "text": "Mathematical illustration",
    "params": []
  },
  "audit.fixedIllustration": {
    "text": "Fixed mathematical illustration {id}",
    "params": [
      "id"
    ]
  },
  "audit.mapping.label": {
    "text": "{side} {field} {number}",
    "params": [
      "side",
      "field",
      "number"
    ]
  },
  "audit.mapping.caption": {
    "text": "{side} {field}",
    "params": [
      "side",
      "field"
    ]
  },
  "audit.mapping.publication": {
    "text": "{side} publication {number}",
    "params": [
      "side",
      "number"
    ]
  },
  "audit.mapping.publicationCaption": {
    "text": "{side} publication ID",
    "params": [
      "side"
    ]
  },
  "correction.side.original": {
    "text": "original",
    "params": []
  },
  "correction.side.replacement": {
    "text": "replacement",
    "params": []
  },
  "correction.field.id": {
    "text": "id",
    "params": []
  },
  "correction.field.version": {
    "text": "version",
    "params": []
  },
  "correction.field.sha256": {
    "text": "sha256",
    "params": []
  },
  "audit.skipped": {
    "text": "Skipped",
    "params": []
  },
  "audit.notPassed": {
    "text": "Not passed",
    "params": []
  },
  "audit.practice": {
    "text": "practice",
    "params": []
  },
  "audit.assessment": {
    "text": "assessment",
    "params": []
  },
  "audit.unsaved": {
    "text": " · Unsaved changes",
    "params": []
  },
  "audit.unsavedQuestion": {
    "text": "· Unsaved changes",
    "params": []
  },
  "audit.legacy": {
    "text": " · Legacy authorship needs verification",
    "params": []
  },
  "audit.none": {
    "text": "None",
    "params": []
  },
  "audit.noneConfigured": {
    "text": "None configured",
    "params": []
  },
  "audit.coverFound": {
    "text": "Five-question cover found",
    "params": []
  },
  "audit.coverNeedsWork": {
    "text": "Five-question cover needs work",
    "params": []
  },
  "audit.objective.label": {
    "text": "{label} objective index {number}",
    "params": [
      "label",
      "number"
    ]
  },
  "audit.source.title": {
    "text": "Source title unverified",
    "params": []
  },
  "audit.source.author": {
    "text": "Author unverified",
    "params": []
  },
  "audit.source.license": {
    "text": "License unverified",
    "params": []
  },
  "audit.source.attribution": {
    "text": "Attribution unverified",
    "params": []
  },
  "layout.description": {
    "text": "Explore a connected map of mathematics, from foundations to new frontiers.",
    "params": []
  },
  "question.state.editing": {
    "text": "editing",
    "params": []
  },
  "question.state.submitted": {
    "text": "submitted",
    "params": []
  },
  "question.state.pending": {
    "text": "pending",
    "params": []
  },
  "question.state.approved": {
    "text": "approved",
    "params": []
  },
  "question.state.returned": {
    "text": "returned",
    "params": []
  },
  "question.state.prepared": {
    "text": "prepared",
    "params": []
  },
  "question.state.published": {
    "text": "published",
    "params": []
  },
  "question.state.active": {
    "text": "active",
    "params": []
  },
  "correction.plan.draft": {
    "text": "draft",
    "params": []
  },
  "correction.plan.pending": {
    "text": "pending",
    "params": []
  },
  "correction.plan.approved": {
    "text": "approved",
    "params": []
  },
  "correction.plan.rejected": {
    "text": "rejected",
    "params": []
  },
  "correction.jobType.withdrawal_impact": {
    "text": "withdrawal impact",
    "params": []
  },
  "correction.jobType.rule_impact": {
    "text": "rule impact",
    "params": []
  },
  "correction.jobType.approved_plan": {
    "text": "approved plan",
    "params": []
  },
  "correction.jobType.attempt_terminal": {
    "text": "attempt terminal",
    "params": []
  },
  "correction.jobState.queued": {
    "text": "queued",
    "params": []
  },
  "correction.jobState.running": {
    "text": "running",
    "params": []
  },
  "correction.jobState.succeeded": {
    "text": "succeeded",
    "params": []
  },
  "correction.jobState.retry_wait": {
    "text": "retry_wait",
    "params": []
  },
  "correction.jobState.failed": {
    "text": "failed",
    "params": []
  },
  "content.field.kind": {
    "text": "kind",
    "params": []
  },
  "learning.attempt.active": {
    "text": "active",
    "params": []
  },
  "learning.attempt.submitted": {
    "text": "submitted",
    "params": []
  },
  "learning.attempt.answered": {
    "text": "answered",
    "params": []
  },
  "learning.attempt.revealed": {
    "text": "revealed",
    "params": []
  },
  "learning.attempt.abandoned": {
    "text": "abandoned",
    "params": []
  },
  "learning.attempt.expired": {
    "text": "expired",
    "params": []
  },
  "learning.attempt.started": {
    "text": "started",
    "params": []
  },
  "learning.attempt.completed": {
    "text": "completed",
    "params": []
  },
  "learning.attempt.enrolled": {
    "text": "enrolled",
    "params": []
  }
} as const;
