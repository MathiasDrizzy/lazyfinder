# QA matrix

Every function of lazyfinder and the test or scenario that proves it. A function with no test is written down as
**NOT TESTED**. The names are real: `go test ./internal/<pkg> -run <Name>` runs a test, and
`tuiqa run testdata/tuiqa/<file>.json` runs a scenario against the built binary in a pty. A Go test
(`TestQAMatrixNamesExist`) fails the gate when a name in this table does not exist.

Legend: **T** = Go test in `internal/ui` unless a package is named; **S** = tuiqa scenario in `testdata/tuiqa/`.

## Keys (one row per action; the second key of an action is pressed too)

| Action (keys) | Proof |
|---|---|
| Up, Down (↑ ↓ k j), First, Last (home end g G) | T KeyboardNavigation, T PageKeysHomeAndEndMoveTheSelection, T UpOnTheFirstRowGoesToTheLastAndDownOnTheLastToTheFirst, S circular |
| PageUp, PageDown (pgup pgdown ctrl+u ctrl+d) | T PageKeysHomeAndEndMoveTheSelection |
| Open, Parent (enter → l / backspace ← h) | T KeyboardNavigation, T EnterOnAChildFolderGoesIntoIt, T BackspaceNeverLeavesTheProjectRoot |
| Back, Forward (alt+← alt+→ [ ]) | T BackForwardButtonsAndKeys, T BracketsGoBackAndForward |
| Preview (space), QuickLook (Q) | T SpacePreviewShowsTextAndClosesWithEsc, T ImagePreviewWithoutSupportOffersQuickLook, T ImagePreviewInsideTmuxSaysTmuxDoesNotPassImages, S preview |
| OpenWith (O) | T OpenWithMenuListsChoicesAndRunsSelection, T OpenWithMenuEscCancels, S open-with |
| Sort, SortReverse (s S) | T SortKeys, T HeaderClickSortsAndReverses |
| Hidden (.), Ignored (I), Reload (r) | T HiddenToggle, T HideIgnoredWithTheKey, T SettingsToggleHiddenReloads |
| Search (/ ctrl+f) | T SlashOpensTheSearchBarAndTypingFiltersTheFolder, T TheOtherKeysOfAnActionDoTheSame, S search |
| Goto (ctrl+g) | T CtrlGOpensAPathPromptAndEnterGoesThere, T TabCompletesAFolderName |
| Expand, Collapse (+ = −) | T PlusExpandsTheFolderInPlaceAndMinusCollapsesIt, T TheOtherKeysOfAnActionDoTheSame, S tree |
| Mark, MarkAll (v ctrl+a) | T VMarksTheRowAndMovesDown, T CtrlAMarksEverythingAndEscClears |
| Copy, Cut, Paste (c x p ctrl+v) | T CopyThenPasteIntoAnotherFolderKeepsTheOriginal, T CutThenPasteMovesAndEmptiesTheClipboard, T CtrlVPastesLikeP |
| Duplicate (D) | T DKeyDuplicates |
| Rename (f2 R) | T RenameWithF2PrefillsTheNameAndRenames, T TheOtherKeysOfAnActionDoTheSame |
| NewFolder, NewFile (N ctrl+n / n) | T NewFolderAsksForANameAndSelectsIt, T ActionIconsCreateFilesAndFoldersAndCollapse, S rebound-keys |
| Trash, Undo (d delete / u ctrl+z) | T TrashOfOneItemAsksNothingAndUndoBringsItBack, T DeleteKeyTrashesToo, T UndoReversesARename |
| Redo (U ctrl+r), History (H) | T URedoesAndHListsTheHistory, T HWithoutHistory, T ARedoRunsApartFromTheUI, fileops: T EveryOperationCanBeRedone |
| CopyPath (y) | T YCopiesThePathOfTheSelectionOrOfTheMarkedRows |
| Chmod (P) | T PSetsThePermissionsOfTheSelection, T ABadModeKeepsThePromptOpen, fileops: T UndoOfAChmodDoesNotFollowALinkPutInPlace |
| Info (i) | T InfoPopupShowsDetailsAndEscClosesIt, T GetInfoFitsTheNarrowestSidebar |
| Layout (M) | T ColumnLayoutToggleReturnsToTheList |
| TagKey, Tags (t T) | T TagToggleOnTheSelectionAndSearchByIt, T TagPickerSearchesTheChosenTag |
| SavedSearches (F), Filter (f) | T ASearchCanBeSavedAndRunAgainFromThePicker, T QuickFilterNarrowsTheFolderWhileTyping |
| Commands (!), Recents (E) | T CommandsPickerRunsTheChosenCommandOnTheSelectionInItsFolder, T RecentsListVisitedFoldersNewestFirstAndOpenOne |
| Bookmark (b) | T BookmarkAddsTheSelectedFolderToFavoritesAndTogglesItOff |
| SetMark, JumpMark (m ') | T VimMarksGoBackToAFolder, T MarkCancelledByAnotherKey |
| Palette (: ctrl+p) | T ThePaletteFindsAndRunsAnAction, T ThePaletteWithNoMatch, T AClickInThePaletteRunsTheItem, T EveryPaletteActionRuns |
| NewTab, CloseTab, NextTab, PrevTab | T NewTabStartsInTheSameFolderAndShowsTheCount, T ClosingATabReturnsToANeighbourAndTheLastOneStays, T TabsWrapAround |
| SidebarNarrow/Widen (< >), ColumnNarrow/Widen (ctrl+←/→), FavoritesShorter/Taller (ctrl+↑/↓) | T KeysResizeTheSidebar, T KeysResizeTheSortedColumn, T TheBoxesCanBeResizedFromTheKeyboard |
| Help (? f1), Settings (,), Focus (tab) | T QuestionMarkOpensHelpThatExplainsFavorites, T TheOtherKeysOfAnActionDoTheSame, T SettingsPopupOpensWithKeyAndGearAndCloses, T TabMovesFocusToSidebar |
| Project (ctrl+b), Branch (B), Quit (q) | T CtrlBTurnsTheProjectSidebarOnAndOff, T TheBranchMenuSwitchesCreatesAndWarnsAboutChanges, T QuitReturnsQuitCmd, T QuitRestoresTheTerminal |
| Every default key is what the test helper presses | T KeyHelperPressesTheKeyItIsAskedFor |
| Rebinding a key | T RebindingAKeyMovesTheAction, S rebound-keys |

## Mouse, hover and tooltips

| Function | Proof |
|---|---|
| Click selects, double click opens | T MouseClickSelectsAndDoubleClickEnters, T ReachNestedFolderWithClicksOnly, S clicks |
| Breadcrumb, back/forward buttons, sidebar places | T BreadcrumbSegmentsAreClickable, T BackForwardButtonsAndKeys, T SidebarClickAndMissingPlace |
| Right click on a breadcrumb segment copies its path (tool, OSC 52 without one, error) | T RightClickOnABreadcrumbCopiesThatFolder, T RightClickOnTheLastBreadcrumbCopiesTheCurrentFolder, T ACopyThatFailsSaysSo, T WithoutAClipboardToolTheTerminalTakesTheCopy, T CopyPipesTheTextIntoTheFirstToolFound, T PbcopyGoesBeforeTheOthers, T WithoutAToolCopySaysThereIsNone, S copy-path |
| Header click sorts; right click chooses columns | T HeaderClickSortsAndReverses, T RightClickOnTheHeaderChoosesColumns, S columns-menu |
| Drag the sidebar border, a column border, the box border, the preview border | T DraggingTheSidebarBorderResizesItAndSavesOnRelease, T DraggingAColumnBorderResizesThatColumn, T DraggingTheBorderBetweenTheBoxesResizesFavoritesAndIsRemembered, T DraggingThePreviewBorderOfTheColumnViewResizesIt |
| Wheel scrolls the list, the Git tab and the diff | T WheelScrollsList, S sidebar-git |
| Ctrl-click marks | T CtrlClickMarksWithoutOpeningAnything |
| Hover paints the row, button or chip under the pointer | T HoverPaintsTheRowUnderThePointerAndKeysTakeItOff, T HoverOnACutSearchChipStaysInsideTheTerminal, S hover |
| Tooltip of every icon and button | T TheTooltipNamesTheIconUnderThePointer, T ButtonsShowOnlyOnTheSelectedOrHoveredRowAndHaveTooltips, S sidebar-actions |
| Mouse off ignores clicks, wheel and hover | T MouseOffIgnoresClicksAndWheel, T NoMouseTurnsHoverOff, S no-mouse |
| Right click menus (favorites, folders, changed files) | T RightClickOnAFavoriteOffersRemoveAndOnAFolderOffersAdd, T RightClickOnAChangedFileOffersTheGitActions, T RightClickOnACleanFileKeepsTheOpenWithMenu |
| Popups close only with Esc | T PopupsCloseOnlyWithEscNotWithAClickOutside, T PopupsShowTheEscHintInTheirBottomBorderLikeLazymark |
| Circular navigation | T UpOnTheFirstRowGoesToTheLastAndDownOnTheLastToTheFirst, T WrapKeepsACursorInsideTheList, S circular |

## Files and folders

| Function | Proof |
|---|---|
| List, folders first, markers, columns, density | T InitialScreenShowsListColumnsSidebarBreadcrumb, T FolderMarkerAndFoldersFirst, T TheDefaultDensityIsWideLikeFinder, S columns |
| New folder or file made at once, its name selected on its own row: typing replaces it, enter keeps it, esc takes it back; the prompt says what is made; no tooltip over it | T ANewFolderAppearsAtOnceAndTypingReplacesItsName, T EscapeTakesBackTheNewFolderOrFile, T EnterKeepsTheDefaultName, T AnArrowKeyKeepsTheDefaultNameToEdit, T AClickOnTheNewFolderIconShowsThePromptNotTheTooltip, T TheNewFileAndNewFolderPromptsDiffer, S new-item, S new-item-sidebar |
| Column separators, choosing columns, narrow windows | T ColumnsAreSeparatedByABarLikeFinder, T ColumnsSettingCyclesBasicFullAndCustom, T NarrowWindowDropsWholeColumnsNeverCutsText, S column-separators |
| Tree (expand in place, survive leaving, live refresh) | T ExpansionSurvivesLeavingAndComingBack, T ExpandedFoldersAreWatchedAndRefreshLive, S tree |
| Column (Miller) layout | T ColumnLayoutShowsParentCurrentAndAPreviewOfTheSelection, T ClickingTheParentColumnGoesThere |
| Copy, cut, paste, duplicate, rename, new, trash, undo, confirmation | T CopyOfMarkedItemsPastesThemAll, T RenameToATakenNameShowsTheErrorAndChangesNothing, T TrashOfSeveralItemsAsksFirst, T ConfirmDeleteOffTrashesWithoutAsking, T ACopyDrawsAProgressBar, T CancellingStaysOnTheStatusLine, S file-ops |
| The session never writes to the tree by itself | T SessionNeverWritesToTheTree |
| Tags, favorites, recents, saved searches, tabs | T TagsFollowARenameAndPersist, T SidebarKeysRemoveRenameAndReorderAFavorite, T RecentsSurviveARestart, T SavedSearchesSurviveARestart, T TabsKeepTheirOwnFolderAndHistory |
| Goto prompt | T ANonexistentPathShowsAnErrorAndKeepsThePrompt, T AFilePathGoesToItsFolderAndSelectsIt |
| Unreadable folder | T UnreadableFolderShowsErrorAndStays, T AFolderWithoutPermissionSaysSoAndStays |
| Odd names (long, CJK, emoji, tab, escape codes, shell syntax) | T OddNamesDoNotBreakTheScreen |
| Broken and looping symlinks | T BrokenSymlinksAreListedAndSafe |
| Empty folder | T AnEmptyFolderShowsAndDoesNotFallOver, S errors |
| Folder of 3000 files | T ABigFolderScrollsAndStaysResponsive, T ANewFileAppearsInUnder300msEvenInAFolderOfThousands |

## Preview, search, open with

| Function | Proof |
|---|---|
| Text, folder, archive, binary, markdown, video, PDF, image previews | T SpacePreviewShowsTextAndClosesWithEsc, T SpaceOnAnArchiveListsItsContents, T SpaceOnABinaryFileShowsAHexDump, T SpaceOnMarkdownShowsItRendered, T SpaceOnAVideoShowsTheProbedFacts, T PDFPageIsDrawnAsAnImageAndTheTempPageIsKept, S previews-more, S preview-c2pa-jpeg, T ACSVShowsAsATable, T ATableWithTooManyColumnsStaysSmall |
| Search by name, content, kind, scope, date, size | T RecursiveSearchFindsDeepFiles, T ContentSearchInTheFolderFindsFilesByTheirText, T FilterChipsAndQueryFilterByDateAndSize, T KindsKeyNarrowsToFilesOrFolders, S search-field |
| Search chips and the bar | T EachSearchFilterButtonCanBeClicked, T SearchFiltersNeverGetCut |
| Open with, commands, openers, terminal handover | T OpenFileUsesConfiguredOpener, T OpenFileWithoutRuleUsesSystemOpen, T TerminalOpenerHandsOverTheTerminal |

## Look, settings, languages

| Function | Proof |
|---|---|
| Every button is herdr's chip ` g ` with the glyph in the middle, the actions side by side, one free cell between groups, the same dimmed color in both modes; the pointer lights it with the theme's surface | T TheButtonBoxesAreHerdrChipsWithTheGlyphInTheMiddle, T TheLitBoxIsCenteredOnTheInkOfANerdGlyphAndAllBoxesAreAlike, T TheTallChipOfTheSidebarHasStraightSides, T TheButtonsHaveTheSameColorInBothModes, S chips, S chips-sidebar |
| Quitting leaves the shell in the folder lazyfinder ends in (`--shell-init` function and `--cwd-file`), and Settings turns it off (`cd_on_exit`) | T TheCwdFileGetsTheFolderTheAppEndsIn, T TheCwdFileStaysEmptyWhenCdOnExitIsOff, T TheShellFunctionMovesTheShellToTheLastFolder, T TheCdOnExitSettingTurnsItOffAndSavesIt, T ExitDirIsTheFolderShownUnlessCdOnExitIsOff, S cd-on-exit |
| The Basic columns are the default ones with the date created; Full leaves out the extension, which goes first on a narrow window | T ColumnPresets, T TheExtensionGoesBeforeTheDateCreatedOnANarrowWindow |
| Row height comes from the density, not from git: inside a repository `normal` rows sit on consecutive lines and the default `comfortable` rows are two lines apart | S rows-git-normal, S rows-git-comfortable, T TheDefaultDensityIsWideLikeFinder |
| Hover and click on every action icon of the window toolbar (the new folder and new file icons open their prompt) | S chips |
| Every part follows the theme (file icons, folders, selection, border, branch, hover); no color is written outside the theme package | T EveryPartChangesColorWithTheTheme, T NoColorIsWrittenOutsideTheThemePackage, S chips-catppuccin-latte, S chips-nord |
| 15 themes readable, light and dark | T TextReadsInEveryTheme, T PlainTextTakesTheThemeTextColorOnALightTheme, S themes |
| Settings (every row saves; a broken config is not overwritten) | T SettingsThemeChangeAppliesAndIsSaved, T SettingsDoesNotOverwriteABrokenConfig, T SettingsFitsAndScrollsAtSmallSizes, S settings |
| Languages (6 catalogs) | T SettingsLanguageChangesTheTextAtOnce, T SidebarAndTitlesUseTheInterfaceLanguage, S language, S language-ja |
| No color / terminal theme / icons off | T NoColorHasNoColorSequences, T TerminalThemeUsesNoTrueColor, T IconsCanBeSwitchedToAsciiOrOff, S no-color |
| Too small terminal | T TooSmallTerminal, S sidebar-small-20 |
| Terminal restored on exit and on panic | T QuitRestoresTheTerminal, T APanicRestoresTheTerminal |

## Sidebar mode

| Function | Proof |
|---|---|
| Enter and leave (ctrl+b, the mode button, `--follow`) | T CtrlBTurnsTheProjectSidebarOnAndOff, T TheModeButtonSwitchesToTheSidebarModeAndBack, S mode-button, S follow-mode |
| Window and sidebar switch by themselves when resized (below 60 columns, from 62 up) | T TheModeChangesByItselfWhenTheWindowIsResizedAcrossTheLimits, T TheModeTheProgramStartedInStaysUntilTheFirstResizeAcrossALimit |
| Buttons of equal width, every cell clickable; no refresh button; icon-only tabs with tooltips; click on a folder row opens it | T EveryCellOfAnActionButtonRunsIt, T ThereIsNoRefreshButtonBecauseTheListUpdatesByItself, T TheTabsShowOnlyTheirIconAndEachHasATooltip, T AClickAnywhereOnAFolderRowOpensAndClosesItInTheTree |
| Every top-row glyph lies inside its own click and hover zone, in both modes and 14 widths | T EveryTopRowGlyphLiesInsideItsOwnZone, T EachGlyphIsDrawnInTheMiddleCellOfItsButton |
| Tree buttons show with the pointer and fade; the name is not repeated; a narrow start is the sidebar | T TheTreeButtonsAppearWithThePointerAndFadeAfterItStops, T TheWindowToolbarKeepsItsButtonsAlways, T TheTreeHeaderAndTheBorderDoNotRepeatTheProjectName, T AWindowThatStartsNarrowerThanTheLimitIsTheSidebar |
| Sidebar drawn like herdr: tall tab bar, every row of a tab takes the click, no frame, triangle rows, the selection keeps the icon color | T TheSidebarTabBarIsThreeRowsTallLikeHerdr, T EachRowOfATabButtonOpensIt, T TheSidebarHasNoFrameOfItsOwn, T TheSidebarRowsAreTheIndentTheTriangleTheIconAndTheName, T TheSelectedRowKeepsTheColorOfItsIcon |
| Detached HEAD with its short hash; no git buttons outside a repository | T ADetachedHeadShowsTheShortHash, T OutsideARepositoryTheGitTabHasNoGitButtons |
| Header buttons of the tree are drawn inside their zones | T EveryHeaderButtonIsDrawnInsideItsZone |
| No scenario checks the screen right after a git key | T NoScenarioAssertsRightAfterAGitOperation |
| The last visible row of the list takes the click | T TheLastVisibleRowOfTheListTakesTheClick |
| Hover keeps the line width over wide glyphs; `.gitignore` keeps its red | T HoverKeepsTheLineWidthOverWideGlyphs, icons: TestAHiddenGitFileKeepsItsRedColor |
| Tabs Files / Search / Git, clickable, even with only glyphs | T ProjectModeHasClickableFilesSearchAndGitTabs, T TheGitTabInEachRepositoryState |
| Tree with actions (new file, new folder, refresh, collapse) | T ActionIconsCreateFilesAndFoldersAndCollapse, S sidebar-actions |
| Git letters and colors per file and folder | T GitLettersAreVSCodesAndColored, T GitColorsTheNamesAndMarksFoldersWithADotAndACounter, S sidebar-git |
| Real time (new file under 300 ms, commit and checkout refresh) | T ANewFileAppearsInUnder300msEvenInAFolderOfThousands, T StagingCommittingAndSwitchingBranchRefreshByThemselves, S realtime-sidebar, S realtime-normal, S live-follow |
| New files in a repository: git tells untracked from committed, the + only outside git | T GitTellsApartTwoNewFilesWhenOneIsCommitted, S git-new-vs-committed |
| The sidebar tree always hides .git, as herdr-sidebar | T TheSidebarTreeAlwaysHidesDotGitLikeHerdr, S git-new-vs-committed |
| Without git installed or outside a repository | T WithoutGitInstalledTheListStillWorks, T OutsideARepositoryTheGitTabSaysSo, T OutsideARepositoryThereIsNoGitGutter |
| 24, 30, 36 and 80 columns | T ProjectModeFitsThirtyColumns, T FollowModeFitsANarrowTerminal, S sidebar-mode-30, S sidebar-mode-40 |

## herdr plugin

| Function | Proof |
|---|---|
| A sidebar docks by itself on the left or right of each tab, once, without the focus | T EnsureOpensTheSidebarOnTheLeftAndGivesTheFocusBack, T EnsureOnTheRightNeedsNoSwap, T EnsureIsIdempotent, T EnsureSkipsWhileAnotherHoldsTheLock, T EnsureDoesNothingWithAutoOpenOff |
| Toggle opens, focuses or closes it; closed stays closed in that tab | T ToggleClosesAFocusedSidebarAndEnsureRespectsIt, T ToggleFocusesASidebarThatIsNotFocused |
| Tab of the event, anchor pane, edge pane | T EventTabComesFromTheEventFirst, T TheAnchorIsTheFocusedPaneOfTheTabAndNeverTheSidebar, T TheEdgePaneIsTheTopmostOneOnTheDockSide |
| The sidebar follows the folder of the pane next to it | T TheSidebarFollowsTheFocusedPaneOfItsTab, T TheSidebarFollowsTheFolderOfTheHerdrPane, T ThePaneOfThePluginStartsInTheSidebarAndFollowsHerdr |
| Dock setting and config | T TheHerdrRowPicksTheEdgeOfTheSidebarAndSavesIt, T HerdrDefaultsAndValidation, T HerdrOutsideHerdrSaysSo |

## Git tab

| Function | Proof |
|---|---|
| Staged and unstaged sections, counts, folding | T TheGitTabSplitsStagedFromChangesWithCounts, S git-tab |
| The Git tab of the sidebar only looks: no stage, discard, stash, branch switch, pull or push (Mathias, 2026-10-14) | T TheGitTabOfTheSidebarOnlyShowsWhatChanged, T TheRightClickOfTheSidebarOffersNoGitChanges, S git-tab |
| Diff view | T EnterShowsTheDiffAndEscClosesIt, S git-tab |
| Open file (`o`) and open changes (↗) | T TheOpenButtonShowsTheDiffAndOShortcutOpensTheFile |
| Branch menu of the window: search, switch, create, warning about changes | T TheBranchMenuSwitchesCreatesAndWarnsAboutChanges, T TheBranchMenuRefusesToLoseWorkAndSaysWhy |
| Conflicts are marked | T ConflictsAreMarked |
| Row counter counts the Git rows | T TheCounterOfTheGitTabCountsItsOwnRows |
| Repository in each state (clean, no commits, modified, staged + untracked, detached, conflict, ahead and behind) at 3 sizes | T TheGitTabInEachRepositoryState |
| Git operations (package `git`) | git: TestStashPushListPopRoundTrip, git: TestChangesOfFindsBothSidesOfAFileByAbsolutePath |
| Right-click git actions in the normal window; the branch in the bar | T RightClickOnAChangedFileOffersTheGitActions, T GitStateAndBranchAreShown |
| Not offered on purpose | commit, rebase, merge (open lazygit with `L`) |

## Whole-screen exploration

| Function | Proof |
|---|---|
| Random keys, clicks, right clicks, hover, wheel in both modes at 6 sizes: the screen always fits, nothing panics | T SweepNormalWindowKeepsTheScreenInsideTheTerminal, T SweepSidebarModeKeepsTheScreenInsideTheTerminal |
| Random resizes from 1x1 to 220x60 | T SweepSurvivesResizesInBothModes |
| A wide glyph under the edge of a popup | T OverlayKeepsTheLineWidthWhenAWideGlyphStraddlesTheBox |
| Goldens of the main screens | T GoldenList120x40, T GoldenList80x24, T GoldenList60x16, T GoldenSettings, T GoldenSpanish |

## NOT TESTED

- Real clipboard and real windows: the suite never touches the user's environment, by rule.
- Opening a file with the real macOS `open`: a fake runner stands in; the real call was checked by hand only.
- Network `git fetch/pull/push` against a remote that asks for credentials: prompts are disabled, so this only fails fast; there is no test with a server.
