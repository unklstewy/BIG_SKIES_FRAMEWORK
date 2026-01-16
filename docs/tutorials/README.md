# Big Skies Tutorial Viewer

An interactive, single-page application for viewing markdown tutorials with automatic lesson segmentation, progress tracking, and syntax highlighting.

## Features

### 📚 **Automatic Lesson Parsing**
- Automatically splits markdown files into lessons based on `## Lesson` headers
- Generates a navigable table of contents
- Each lesson displayed separately for focused learning

### ✅ **Progress Tracking**
- Interactive checklist for each lesson
- Progress bar showing completion percentage
- State persisted in browser localStorage (survives page reloads)
- Visual indicators for completed lessons

### 💻 **Code Highlighting**
- Syntax highlighting for Go, Bash, JavaScript, HTML, CSS, and more
- **Comment highlighting** - Code comments automatically highlighted as "learning points of interest"
- Copy-to-clipboard button for all code blocks
- Language labels on code blocks

### 🎨 **Modern UI**
- Three-panel layout: Navigation | Content | Progress
- Responsive design (works on desktop, tablet, mobile)
- Dark code theme with professional styling
- Smooth animations and transitions

### ⌨️ **Keyboard Shortcuts**
- `Arrow Left` - Previous lesson
- `Arrow Right` - Next lesson

### 🔄 **Multi-File Support**
- Dropdown selector for multiple tutorial files
- Automatically discovers markdown files in the directory
- Separate progress tracking for each tutorial

## Usage

### Quick Start

1. **Open the viewer:**
   ```bash
   cd /Volumes/MacStorage/Development/BIG_SKIES_FRAMEWORK/docs/tutorials
   open viewer.html
   ```

2. **Select a tutorial** from the dropdown (defaults to Novice Developer Guide)

3. **Navigate lessons** using:
   - Sidebar lesson menu (click any lesson)
   - Previous/Next buttons at bottom
   - Keyboard arrows (← →)

4. **Track progress** by checking off lessons in the right panel

### Adding New Tutorials

To add a new markdown tutorial file:

1. Place your `.md` file in the `docs/tutorials/` directory

2. Edit `viewer.html` line 816-819 to add your file:
   ```javascript
   const knownFiles = [
       'NOVICE_DEVELOPER_GUIDE.md',
       'YOUR_NEW_TUTORIAL.md',  // Add here
   ];
   ```

3. Reload the page - your tutorial will appear in the dropdown

### Tutorial Markdown Format

The viewer expects markdown structured with:

- `# Title` - Main title (optional, only shown in introduction)
- `## Lesson Title` - Each `##` header creates a new lesson
- Standard markdown formatting
- Code blocks with language specification for highlighting

**Example:**
```markdown
# My Tutorial

Introduction content here...

## Lesson 1: Getting Started

Lesson 1 content...

## Lesson 2: Advanced Topics

Lesson 2 content...
```

## Technical Details

### Dependencies

All dependencies loaded from CDN:
- **marked.js** (4.x) - Markdown parsing
- **highlight.js** (11.9.0) - Code syntax highlighting

### Browser Compatibility

- Modern browsers (Chrome, Firefox, Safari, Edge)
- Requires JavaScript enabled
- Uses ES6+ features (async/await, arrow functions, etc.)

### Local Storage Keys

Progress is stored with keys: `bigskies-tutorial-{filename}-completed`

To reset progress:
```javascript
// In browser console
localStorage.clear();
```

### File Structure

```
docs/tutorials/
├── viewer.html              # Main viewer application (single file)
├── README.md               # This file
├── NOVICE_DEVELOPER_GUIDE.md  # Tutorial content
└── [other .md files]       # Additional tutorials
```

## Features Explained

### Comment Highlighting

The viewer automatically highlights code comments (lines starting with `//`, `#`, `/*`, or `<!--`) with a subtle yellow background. This draws attention to the exhaustive explanations in the tutorial code.

### Progress Persistence

Your progress is automatically saved:
- Checked lessons persist across page reloads
- Separate progress for each tutorial file
- Stored in browser's localStorage

### Lesson Navigation

Three ways to navigate:
1. **Sidebar menu** - Click any lesson to jump directly
2. **Navigation buttons** - Sequential Previous/Next at bottom
3. **Keyboard shortcuts** - Arrow keys for quick navigation

### Code Copy Feature

Hover over any code block to reveal a "Copy" button in the top-right corner. Click to copy the entire code block to clipboard.

## Customization

### Styling

All CSS is contained in the `<style>` tag (lines 16-696). Key customization points:

- **Colors**: CSS variables in `:root` (lines 29-60)
- **Spacing**: Adjust `--spacing-*` variables
- **Fonts**: Change `--font-family` and `--font-mono`

### Behavior

JavaScript functions are well-documented with JSDoc comments. Key functions:

- `parseMarkdownIntoLessons()` - Controls lesson segmentation
- `highlightComments()` - Customize comment detection
- `enhanceCodeBlocks()` - Modify code block enhancements

## Troubleshooting

### "No tutorials found"
- Verify markdown files are in the same directory as `viewer.html`
- Check the `knownFiles` array includes your filename
- Ensure files have `.md` extension

### Progress not saving
- Check browser localStorage is enabled
- Try a different browser
- Clear cache and reload

### Code not highlighting
- Verify code blocks use triple backticks with language: ` ```go `
- Check browser console for errors
- Ensure highlight.js CDN is accessible

### Comments not highlighted
- Comments must start at the beginning of the trimmed line
- Minimum 3 characters to be highlighted
- Supported comment styles: `//`, `#`, `/*`, `*`, `<!--`

## Performance

- **Lazy loading**: Only current lesson rendered
- **Client-side only**: No server required
- **Lightweight**: ~50KB HTML + external CDN libraries
- **Fast**: Instant lesson switching, no page reloads

## Security

- Markdown rendered client-side (no server processing)
- No external data fetching (except CDN libraries)
- localStorage limited to progress tracking
- Safe for local file usage

## Credits

Built for the Big Skies Framework tutorial system using:
- [marked.js](https://marked.js.org/) - Fast markdown parser
- [highlight.js](https://highlightjs.org/) - Syntax highlighting

## License

Part of the Big Skies Framework project.
